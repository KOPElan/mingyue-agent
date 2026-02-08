package indexer

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// FileMetadata represents indexed file metadata
type FileMetadata struct {
	ID           int64     `json:"id"`
	Path         string    `json:"path"`
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	ModTime      time.Time `json:"mod_time"`
	IsDir        bool      `json:"is_dir"`
	MimeType     string    `json:"mime_type,omitempty"`
	MD5Hash      string    `json:"md5_hash,omitempty"`
	ThumbnailURL string    `json:"thumbnail_url,omitempty"`
	IndexedAt    time.Time `json:"indexed_at"`
}

// ScanOptions defines scanning behavior
type ScanOptions struct {
	Paths       []string
	Recursive   bool
	Incremental bool
	Extensions  []string // Filter by file extensions
}

// Indexer handles file scanning and metadata indexing
type Indexer struct {
	db          *sql.DB
	mu          sync.RWMutex
	scanPaths   []string
	lastScanRun time.Time
}

// New creates a new Indexer instance
func New(dbPath string) (*Indexer, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	idx := &Indexer{
		db: db,
	}

	if err := idx.initDB(); err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize database: %w", err)
	}

	return idx, nil
}

func (i *Indexer) initDB() error {
	schema := `
	CREATE TABLE IF NOT EXISTS file_metadata (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		path TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		size INTEGER,
		mod_time INTEGER,
		is_dir INTEGER,
		mime_type TEXT,
		md5_hash TEXT,
		thumbnail_url TEXT,
		indexed_at INTEGER,
		created_at INTEGER DEFAULT (strftime('%s', 'now'))
	);

	CREATE INDEX IF NOT EXISTS idx_path ON file_metadata(path);
	CREATE INDEX IF NOT EXISTS idx_name ON file_metadata(name);
	CREATE INDEX IF NOT EXISTS idx_mod_time ON file_metadata(mod_time);
	CREATE INDEX IF NOT EXISTS idx_mime_type ON file_metadata(mime_type);

	CREATE TABLE IF NOT EXISTS scan_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		scan_path TEXT NOT NULL,
		started_at INTEGER,
		completed_at INTEGER,
		files_scanned INTEGER,
		files_added INTEGER,
		files_updated INTEGER,
		errors INTEGER
	);
	`

	_, err := i.db.Exec(schema)
	return err
}

// Scan performs file scanning according to options
func (i *Indexer) Scan(ctx context.Context, opts ScanOptions) (*ScanResult, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	result := &ScanResult{
		StartedAt: time.Now(),
	}

	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, scanPath := range opts.Paths {
		if err := i.scanPath(ctx, tx, scanPath, opts, result); err != nil {
			result.Errors++
			continue
		}
	}

	result.CompletedAt = time.Now()

	// Record scan history
	_, err = tx.Exec(`
		INSERT INTO scan_history (scan_path, started_at, completed_at, files_scanned, files_added, files_updated, errors)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, filepath.Join(opts.Paths...), result.StartedAt.Unix(), result.CompletedAt.Unix(),
		result.FilesScanned, result.FilesAdded, result.FilesUpdated, result.Errors)
	if err != nil {
		return nil, fmt.Errorf("record scan history: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	i.lastScanRun = result.CompletedAt
	return result, nil
}

type ScanResult struct {
	StartedAt    time.Time `json:"started_at"`
	CompletedAt  time.Time `json:"completed_at"`
	FilesScanned int       `json:"files_scanned"`
	FilesAdded   int       `json:"files_added"`
	FilesUpdated int       `json:"files_updated"`
	Errors       int       `json:"errors"`
}

// Stats summarizes indexer metadata for diagnostics.
type Stats struct {
	TotalFiles int
	TotalSize  int64
	LastScan   time.Time
}

// Stats returns aggregate statistics from the indexer database.
func (i *Indexer) Stats(ctx context.Context) (*Stats, error) {
	result := &Stats{}

	if err := i.db.QueryRowContext(ctx, "SELECT COUNT(*), COALESCE(SUM(size), 0) FROM file_metadata").Scan(&result.TotalFiles, &result.TotalSize); err != nil {
		return nil, fmt.Errorf("query file stats: %w", err)
	}

	var lastScanUnix sql.NullInt64
	if err := i.db.QueryRowContext(ctx, "SELECT completed_at FROM scan_history ORDER BY completed_at DESC LIMIT 1").Scan(&lastScanUnix); err != nil {
		return nil, fmt.Errorf("query scan history: %w", err)
	}
	if lastScanUnix.Valid {
		result.LastScan = time.Unix(lastScanUnix.Int64, 0)
	}

	return result, nil
}

func (i *Indexer) scanPath(ctx context.Context, tx *sql.Tx, path string, opts ScanOptions, result *ScanResult) error {
	return filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			result.Errors++
			return nil
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Skip if not recursive and not in root directory
		if !opts.Recursive && filepath.Dir(filePath) != path {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		result.FilesScanned++

		// Check if incremental and file hasn't changed
		if opts.Incremental {
			var lastModTime int64
			err := tx.QueryRow("SELECT mod_time FROM file_metadata WHERE path = ?", filePath).Scan(&lastModTime)
			if err == nil && lastModTime == info.ModTime().Unix() {
				return nil
			}
		}

		// Index the file
		metadata := &FileMetadata{
			Path:      filePath,
			Name:      info.Name(),
			Size:      info.Size(),
			ModTime:   info.ModTime(),
			IsDir:     info.IsDir(),
			IndexedAt: time.Now(),
		}

		// Calculate MD5 for regular files
		if !info.IsDir() && info.Size() < 100*1024*1024 { // Limit to 100MB
			if hash, err := calculateMD5(filePath); err == nil {
				metadata.MD5Hash = hash
			}
		}

		// Detect MIME type
		metadata.MimeType = detectMimeType(filePath)

		// Insert or update
		_, err = tx.Exec(`
			INSERT INTO file_metadata (path, name, size, mod_time, is_dir, mime_type, md5_hash, indexed_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(path) DO UPDATE SET
				name = excluded.name,
				size = excluded.size,
				mod_time = excluded.mod_time,
				mime_type = excluded.mime_type,
				md5_hash = excluded.md5_hash,
				indexed_at = excluded.indexed_at
		`, metadata.Path, metadata.Name, metadata.Size, metadata.ModTime.Unix(),
			metadata.IsDir, metadata.MimeType, metadata.MD5Hash, metadata.IndexedAt.Unix())

		if err != nil {
			result.Errors++
		} else {
			result.FilesAdded++
		}

		return nil
	})
}

// Search searches indexed files by query
func (i *Indexer) Search(ctx context.Context, query string, limit, offset int) ([]*FileMetadata, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	rows, err := i.db.QueryContext(ctx, `
		SELECT id, path, name, size, mod_time, is_dir, mime_type, md5_hash, thumbnail_url, indexed_at
		FROM file_metadata
		WHERE name LIKE ? OR path LIKE ?
		ORDER BY indexed_at DESC
		LIMIT ? OFFSET ?
	`, "%"+query+"%", "%"+query+"%", limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*FileMetadata
	for rows.Next() {
		var m FileMetadata
		var modTime, indexedAt int64
		var isDir int

		err := rows.Scan(&m.ID, &m.Path, &m.Name, &m.Size, &modTime, &isDir,
			&m.MimeType, &m.MD5Hash, &m.ThumbnailURL, &indexedAt)
		if err != nil {
			continue
		}

		m.ModTime = time.Unix(modTime, 0)
		m.IndexedAt = time.Unix(indexedAt, 0)
		m.IsDir = isDir != 0

		results = append(results, &m)
	}

	return results, rows.Err()
}

// GetByPath retrieves file metadata by path
func (i *Indexer) GetByPath(path string) (*FileMetadata, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	var m FileMetadata
	var modTime, indexedAt int64
	var isDir int

	err := i.db.QueryRow(`
		SELECT id, path, name, size, mod_time, is_dir, mime_type, md5_hash, thumbnail_url, indexed_at
		FROM file_metadata
		WHERE path = ?
	`, path).Scan(&m.ID, &m.Path, &m.Name, &m.Size, &modTime, &isDir,
		&m.MimeType, &m.MD5Hash, &m.ThumbnailURL, &indexedAt)
	if err != nil {
		return nil, err
	}

	m.ModTime = time.Unix(modTime, 0)
	m.IndexedAt = time.Unix(indexedAt, 0)
	m.IsDir = isDir != 0

	return &m, nil
}

// UpdateThumbnailURL updates the thumbnail URL for a file
func (i *Indexer) UpdateThumbnailURL(path, thumbnailURL string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	_, err := i.db.Exec("UPDATE file_metadata SET thumbnail_url = ? WHERE path = ?", thumbnailURL, path)
	return err
}

// CleanupOrphans removes entries for non-existent files
func (i *Indexer) CleanupOrphans(ctx context.Context) (int, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	rows, err := i.db.QueryContext(ctx, "SELECT path FROM file_metadata")
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var toDelete []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			continue
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			toDelete = append(toDelete, path)
		}
	}

	if len(toDelete) == 0 {
		return 0, nil
	}

	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	for _, path := range toDelete {
		_, err := tx.Exec("DELETE FROM file_metadata WHERE path = ?", path)
		if err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return len(toDelete), nil
}

// Close closes the database connection
func (i *Indexer) Close() error {
	return i.db.Close()
}

func calculateMD5(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func detectMimeType(filePath string) string {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".mp4":
		return "video/mp4"
	case ".pdf":
		return "application/pdf"
	case ".txt":
		return "text/plain"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	default:
		return "application/octet-stream"
	}
}
