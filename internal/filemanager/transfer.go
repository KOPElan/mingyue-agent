package filemanager

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

type UploadOptions struct {
	Path          string
	TempDir       string
	MaxSize       int64
	ChunkSize     int64
	ResumeSupport bool
}

type DownloadOptions struct {
	Path       string
	RangeStart int64
	RangeEnd   int64
}

func (m *Manager) Upload(ctx context.Context, reader io.Reader, opts UploadOptions, user string) error {
	if err := m.validator.ValidatePath(opts.Path); err != nil {
		m.logAudit(ctx, user, "upload", opts.Path, "failed", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("invalid path: %w", err)
	}

	dir := filepath.Dir(opts.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		m.logAudit(ctx, user, "upload", opts.Path, "failed", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("create directory: %w", err)
	}

	tempFile := opts.Path + ".tmp"
	f, err := os.Create(tempFile)
	if err != nil {
		m.logAudit(ctx, user, "upload", opts.Path, "failed", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("create temp file: %w", err)
	}
	defer f.Close()

	var written int64
	if opts.MaxSize > 0 {
		limited := io.LimitReader(reader, opts.MaxSize)
		written, err = io.Copy(f, limited)
	} else {
		written, err = io.Copy(f, reader)
	}

	if err != nil {
		os.Remove(tempFile)
		m.logAudit(ctx, user, "upload", opts.Path, "failed", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("write file: %w", err)
	}

	if err := f.Close(); err != nil {
		os.Remove(tempFile)
		m.logAudit(ctx, user, "upload", opts.Path, "failed", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("close file: %w", err)
	}

	if err := os.Rename(tempFile, opts.Path); err != nil {
		os.Remove(tempFile)
		m.logAudit(ctx, user, "upload", opts.Path, "failed", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("rename file: %w", err)
	}

	m.logAudit(ctx, user, "upload", opts.Path, "success", map[string]interface{}{"size": written})
	return nil
}

func (m *Manager) Download(ctx context.Context, writer io.Writer, opts DownloadOptions, user string) (int64, error) {
	if err := m.validator.ValidatePath(opts.Path); err != nil {
		m.logAudit(ctx, user, "download", opts.Path, "failed", map[string]interface{}{"error": err.Error()})
		return 0, fmt.Errorf("invalid path: %w", err)
	}

	f, err := os.Open(opts.Path)
	if err != nil {
		m.logAudit(ctx, user, "download", opts.Path, "failed", map[string]interface{}{"error": err.Error()})
		return 0, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	var reader io.Reader = f

	if opts.RangeStart > 0 {
		if _, err := f.Seek(opts.RangeStart, 0); err != nil {
			m.logAudit(ctx, user, "download", opts.Path, "failed", map[string]interface{}{"error": err.Error()})
			return 0, fmt.Errorf("seek file: %w", err)
		}
	}

	if opts.RangeEnd > 0 {
		reader = io.LimitReader(f, opts.RangeEnd-opts.RangeStart+1)
	}

	written, err := io.Copy(writer, reader)
	if err != nil {
		m.logAudit(ctx, user, "download", opts.Path, "failed", map[string]interface{}{"error": err.Error()})
		return written, fmt.Errorf("copy file: %w", err)
	}

	m.logAudit(ctx, user, "download", opts.Path, "success", map[string]interface{}{
		"size":        written,
		"range_start": opts.RangeStart,
		"range_end":   opts.RangeEnd,
	})

	return written, nil
}

func (m *Manager) GetChecksum(ctx context.Context, path string, user string) (string, error) {
	if err := m.validator.ValidatePath(path); err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", fmt.Errorf("compute hash: %w", err)
	}

	checksum := fmt.Sprintf("%x", hash.Sum(nil))
	m.logAudit(ctx, user, "checksum", path, "success", map[string]interface{}{"checksum": checksum})

	return checksum, nil
}

func ParseRangeHeader(rangeHeader string, fileSize int64) (start, end int64, err error) {
	if rangeHeader == "" {
		return 0, 0, nil
	}

	var rangeStart, rangeEnd string
	if _, err := fmt.Sscanf(rangeHeader, "bytes=%s-%s", &rangeStart, &rangeEnd); err != nil {
		return 0, 0, fmt.Errorf("invalid range header")
	}

	start = 0
	if rangeStart != "" {
		start, err = strconv.ParseInt(rangeStart, 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid range start")
		}
	}

	end = fileSize - 1
	if rangeEnd != "" {
		end, err = strconv.ParseInt(rangeEnd, 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid range end")
		}
	}

	if start > end || start < 0 || end >= fileSize {
		return 0, 0, fmt.Errorf("invalid range")
	}

	return start, end, nil
}
