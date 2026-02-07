package thumbnail

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// Config holds thumbnail generation configuration
type Config struct {
	CacheDir      string
	MaxCacheSize  int64 // bytes
	ImageWidth    int
	ImageHeight   int
	VideoWidth    int
	VideoHeight   int
	Quality       int
	CleanupPolicy CleanupPolicy
}

// CleanupPolicy defines cache cleanup behavior
type CleanupPolicy struct {
	MaxAge       time.Duration // Remove thumbnails older than this
	MaxCacheSize int64         // Remove oldest when cache exceeds this
}

// Generator handles thumbnail generation and caching
type Generator struct {
	config     Config
	mu         sync.RWMutex
	cache      map[string]*ThumbnailInfo
	cacheSize  int64
	lastCleanup time.Time
}

// ThumbnailInfo contains thumbnail metadata
type ThumbnailInfo struct {
	SourcePath string    `json:"source_path"`
	ThumbPath  string    `json:"thumb_path"`
	Size       int64     `json:"size"`
	CreatedAt  time.Time `json:"created_at"`
	AccessedAt time.Time `json:"accessed_at"`
	MimeType   string    `json:"mime_type"`
}

// New creates a new thumbnail generator
func New(config Config) (*Generator, error) {
	if config.CacheDir == "" {
		config.CacheDir = "/var/cache/mingyue-agent/thumbnails"
	}
	if config.ImageWidth == 0 {
		config.ImageWidth = 200
	}
	if config.ImageHeight == 0 {
		config.ImageHeight = 200
	}
	if config.VideoWidth == 0 {
		config.VideoWidth = 320
	}
	if config.VideoHeight == 0 {
		config.VideoHeight = 240
	}
	if config.Quality == 0 {
		config.Quality = 85
	}

	if err := os.MkdirAll(config.CacheDir, 0755); err != nil {
		return nil, fmt.Errorf("create cache directory: %w", err)
	}

	g := &Generator{
		config: config,
		cache:  make(map[string]*ThumbnailInfo),
	}

	// Load existing cache
	if err := g.loadCache(); err != nil {
		// Non-fatal, continue
	}

	return g, nil
}

func (g *Generator) loadCache() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	return filepath.Walk(g.config.CacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// Track cache entry
		g.cache[path] = &ThumbnailInfo{
			ThumbPath:  path,
			Size:       info.Size(),
			CreatedAt:  info.ModTime(),
			AccessedAt: info.ModTime(),
		}
		g.cacheSize += info.Size()

		return nil
	})
}

// Generate creates a thumbnail for the given file
func (g *Generator) Generate(ctx context.Context, sourcePath string) (*ThumbnailInfo, error) {
	// Check if thumbnail already exists
	if info := g.getCached(sourcePath); info != nil {
		g.updateAccessTime(sourcePath)
		return info, nil
	}

	// Determine file type
	mimeType := detectMimeType(sourcePath)
	var thumbPath string
	var err error

	switch {
	case isImage(mimeType):
		thumbPath, err = g.generateImageThumbnail(ctx, sourcePath)
	case isVideo(mimeType):
		thumbPath, err = g.generateVideoThumbnail(ctx, sourcePath)
	case isDocument(mimeType):
		thumbPath, err = g.generateDocumentThumbnail(ctx, sourcePath)
	default:
		return nil, fmt.Errorf("unsupported file type: %s", mimeType)
	}

	if err != nil {
		return nil, err
	}

	info, err := os.Stat(thumbPath)
	if err != nil {
		return nil, err
	}

	thumbInfo := &ThumbnailInfo{
		SourcePath: sourcePath,
		ThumbPath:  thumbPath,
		Size:       info.Size(),
		CreatedAt:  time.Now(),
		AccessedAt: time.Now(),
		MimeType:   mimeType,
	}

	g.addToCache(sourcePath, thumbInfo)

	// Trigger cleanup if needed
	if g.shouldCleanup() {
		go g.Cleanup(context.Background())
	}

	return thumbInfo, nil
}

func (g *Generator) generateImageThumbnail(ctx context.Context, sourcePath string) (string, error) {
	thumbPath := g.getThumbnailPath(sourcePath, ".jpg")

	// Use ImageMagick/convert if available
	cmd := exec.CommandContext(ctx, "convert",
		sourcePath,
		"-thumbnail", fmt.Sprintf("%dx%d>", g.config.ImageWidth, g.config.ImageHeight),
		"-quality", fmt.Sprintf("%d", g.config.Quality),
		thumbPath)

	if err := cmd.Run(); err != nil {
		// Fallback: just copy the original (simplified)
		return g.copyAsThumb(sourcePath, thumbPath)
	}

	return thumbPath, nil
}

func (g *Generator) generateVideoThumbnail(ctx context.Context, sourcePath string) (string, error) {
	thumbPath := g.getThumbnailPath(sourcePath, ".jpg")

	// Use ffmpeg to extract a frame at 1 second
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", sourcePath,
		"-ss", "00:00:01.000",
		"-vframes", "1",
		"-vf", fmt.Sprintf("scale=%d:%d", g.config.VideoWidth, g.config.VideoHeight),
		"-y",
		thumbPath)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffmpeg failed: %w", err)
	}

	return thumbPath, nil
}

func (g *Generator) generateDocumentThumbnail(ctx context.Context, sourcePath string) (string, error) {
	thumbPath := g.getThumbnailPath(sourcePath, ".jpg")

	// Use pdftoppm for PDFs
	if filepath.Ext(sourcePath) == ".pdf" {
		cmd := exec.CommandContext(ctx, "pdftoppm",
			"-jpeg",
			"-f", "1",
			"-l", "1",
			"-scale-to", fmt.Sprintf("%d", g.config.ImageWidth),
			sourcePath,
			thumbPath[:len(thumbPath)-4]) // pdftoppm adds -1.jpg

		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("pdftoppm failed: %w", err)
		}

		// Rename the output
		generatedPath := thumbPath[:len(thumbPath)-4] + "-1.jpg"
		if err := os.Rename(generatedPath, thumbPath); err != nil {
			return "", err
		}

		return thumbPath, nil
	}

	return "", fmt.Errorf("unsupported document type")
}

func (g *Generator) getThumbnailPath(sourcePath, ext string) string {
	base := filepath.Base(sourcePath)
	hash := fmt.Sprintf("%x", md5Sum(sourcePath))
	return filepath.Join(g.config.CacheDir, hash[:8]+"-"+base+ext)
}

func (g *Generator) copyAsThumb(src, dst string) (string, error) {
	input, err := os.ReadFile(src)
	if err != nil {
		return "", err
	}

	err = os.WriteFile(dst, input, 0644)
	if err != nil {
		return "", err
	}

	return dst, nil
}

func (g *Generator) getCached(sourcePath string) *ThumbnailInfo {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, info := range g.cache {
		if info.SourcePath == sourcePath {
			return info
		}
	}

	return nil
}

func (g *Generator) addToCache(sourcePath string, info *ThumbnailInfo) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.cache[sourcePath] = info
	g.cacheSize += info.Size
}

func (g *Generator) updateAccessTime(sourcePath string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if info, ok := g.cache[sourcePath]; ok {
		info.AccessedAt = time.Now()
	}
}

func (g *Generator) shouldCleanup() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Cleanup if cache is too large
	if g.config.MaxCacheSize > 0 && g.cacheSize > g.config.MaxCacheSize {
		return true
	}

	// Cleanup periodically
	if time.Since(g.lastCleanup) > 24*time.Hour {
		return true
	}

	return false
}

// Cleanup removes old or excess thumbnails
func (g *Generator) Cleanup(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.lastCleanup = time.Now()

	var toRemove []string

	// Remove old thumbnails
	if g.config.CleanupPolicy.MaxAge > 0 {
		for path, info := range g.cache {
			if time.Since(info.AccessedAt) > g.config.CleanupPolicy.MaxAge {
				toRemove = append(toRemove, path)
			}
		}
	}

	// Remove excess if cache is too large
	if g.config.MaxCacheSize > 0 && g.cacheSize > g.config.MaxCacheSize {
		// Sort by access time and remove oldest
		// (simplified: just remove randomly until under limit)
		for path, info := range g.cache {
			if g.cacheSize <= g.config.MaxCacheSize {
				break
			}
			toRemove = append(toRemove, path)
			g.cacheSize -= info.Size
		}
	}

	// Actually remove files
	for _, path := range toRemove {
		if info, ok := g.cache[path]; ok {
			os.Remove(info.ThumbPath)
			delete(g.cache, path)
			g.cacheSize -= info.Size
		}
	}

	return nil
}

func isImage(mimeType string) bool {
	return mimeType == "image/jpeg" || mimeType == "image/png" || mimeType == "image/gif"
}

func isVideo(mimeType string) bool {
	return mimeType == "video/mp4" || mimeType == "video/mpeg" || mimeType == "video/quicktime"
}

func isDocument(mimeType string) bool {
	return mimeType == "application/pdf"
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
	default:
		return "application/octet-stream"
	}
}

func md5Sum(s string) []byte {
	// Simplified for now
	return []byte(s)
}
