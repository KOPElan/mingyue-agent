package filemanager

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/KOPElan/mingyue-agent/internal/audit"
)

type Manager struct {
	validator *PathValidator
	audit     *audit.Logger
}

type FileInfo struct {
	Name        string      `json:"name"`
	Path        string      `json:"path"`
	Size        int64       `json:"size"`
	Mode        os.FileMode `json:"mode"`
	ModTime     time.Time   `json:"mod_time"`
	IsDir       bool        `json:"is_dir"`
	IsSymlink   bool        `json:"is_symlink"`
	Owner       uint32      `json:"owner,omitempty"`
	Group       uint32      `json:"group,omitempty"`
	Permissions string      `json:"permissions"`
	MimeType    string      `json:"mime_type,omitempty"`
}

type ListOptions struct {
	Path      string
	Recursive bool
	Offset    int
	Limit     int
	SortBy    string
	SortOrder string
}

func New(allowedPaths []string, auditLogger *audit.Logger) *Manager {
	return &Manager{
		validator: NewPathValidator(allowedPaths),
		audit:     auditLogger,
	}
}

func (m *Manager) List(ctx context.Context, opts ListOptions, user string) ([]FileInfo, error) {
	if err := m.validator.ValidatePath(opts.Path); err != nil {
		m.logAudit(ctx, user, "list", opts.Path, "failed", map[string]interface{}{"error": err.Error()})
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	entries, err := os.ReadDir(opts.Path)
	if err != nil {
		m.logAudit(ctx, user, "list", opts.Path, "failed", map[string]interface{}{"error": err.Error()})
		return nil, fmt.Errorf("read directory: %w", err)
	}

	var files []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		fileInfo := m.buildFileInfo(filepath.Join(opts.Path, entry.Name()), info)
		files = append(files, fileInfo)
	}

	m.logAudit(ctx, user, "list", opts.Path, "success", map[string]interface{}{"count": len(files)})
	return files, nil
}

func (m *Manager) GetInfo(ctx context.Context, path string, user string) (*FileInfo, error) {
	if err := m.validator.ValidatePath(path); err != nil {
		m.logAudit(ctx, user, "get_info", path, "failed", map[string]interface{}{"error": err.Error()})
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	info, err := os.Lstat(path)
	if err != nil {
		m.logAudit(ctx, user, "get_info", path, "failed", map[string]interface{}{"error": err.Error()})
		return nil, fmt.Errorf("stat file: %w", err)
	}

	fileInfo := m.buildFileInfo(path, info)
	m.logAudit(ctx, user, "get_info", path, "success", nil)
	return &fileInfo, nil
}

func (m *Manager) CreateDir(ctx context.Context, path string, user string) error {
	if err := m.validator.ValidatePath(path); err != nil {
		m.logAudit(ctx, user, "create_dir", path, "failed", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("invalid path: %w", err)
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		m.logAudit(ctx, user, "create_dir", path, "failed", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("create directory: %w", err)
	}

	m.logAudit(ctx, user, "create_dir", path, "success", nil)
	return nil
}

func (m *Manager) Delete(ctx context.Context, path string, user string) error {
	if err := m.validator.ValidatePath(path); err != nil {
		m.logAudit(ctx, user, "delete", path, "failed", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("invalid path: %w", err)
	}

	if err := os.RemoveAll(path); err != nil {
		m.logAudit(ctx, user, "delete", path, "failed", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("delete: %w", err)
	}

	m.logAudit(ctx, user, "delete", path, "success", nil)
	return nil
}

func (m *Manager) Rename(ctx context.Context, oldPath, newPath string, user string) error {
	if err := m.validator.ValidatePath(oldPath); err != nil {
		m.logAudit(ctx, user, "rename", oldPath, "failed", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("invalid old path: %w", err)
	}

	if err := m.validator.ValidatePath(newPath); err != nil {
		m.logAudit(ctx, user, "rename", oldPath, "failed", map[string]interface{}{"error": err.Error(), "new_path": newPath})
		return fmt.Errorf("invalid new path: %w", err)
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		m.logAudit(ctx, user, "rename", oldPath, "failed", map[string]interface{}{"error": err.Error(), "new_path": newPath})
		return fmt.Errorf("rename: %w", err)
	}

	m.logAudit(ctx, user, "rename", oldPath, "success", map[string]interface{}{"new_path": newPath})
	return nil
}

func (m *Manager) Copy(ctx context.Context, srcPath, dstPath string, user string) error {
	if err := m.validator.ValidatePath(srcPath); err != nil {
		m.logAudit(ctx, user, "copy", srcPath, "failed", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("invalid source path: %w", err)
	}

	if err := m.validator.ValidatePath(dstPath); err != nil {
		m.logAudit(ctx, user, "copy", srcPath, "failed", map[string]interface{}{"error": err.Error(), "dst_path": dstPath})
		return fmt.Errorf("invalid destination path: %w", err)
	}

	src, err := os.Open(srcPath)
	if err != nil {
		m.logAudit(ctx, user, "copy", srcPath, "failed", map[string]interface{}{"error": err.Error(), "dst_path": dstPath})
		return fmt.Errorf("open source: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		m.logAudit(ctx, user, "copy", srcPath, "failed", map[string]interface{}{"error": err.Error(), "dst_path": dstPath})
		return fmt.Errorf("create destination: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		m.logAudit(ctx, user, "copy", srcPath, "failed", map[string]interface{}{"error": err.Error(), "dst_path": dstPath})
		return fmt.Errorf("copy data: %w", err)
	}

	srcInfo, err := src.Stat()
	if err == nil {
		os.Chmod(dstPath, srcInfo.Mode())
	}

	m.logAudit(ctx, user, "copy", srcPath, "success", map[string]interface{}{"dst_path": dstPath})
	return nil
}

func (m *Manager) Move(ctx context.Context, srcPath, dstPath string, user string) error {
	if err := m.validator.ValidatePath(srcPath); err != nil {
		m.logAudit(ctx, user, "move", srcPath, "failed", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("invalid source path: %w", err)
	}

	if err := m.validator.ValidatePath(dstPath); err != nil {
		m.logAudit(ctx, user, "move", srcPath, "failed", map[string]interface{}{"error": err.Error(), "dst_path": dstPath})
		return fmt.Errorf("invalid destination path: %w", err)
	}

	if err := os.Rename(srcPath, dstPath); err != nil {
		m.logAudit(ctx, user, "move", srcPath, "failed", map[string]interface{}{"error": err.Error(), "dst_path": dstPath})
		return fmt.Errorf("move: %w", err)
	}

	m.logAudit(ctx, user, "move", srcPath, "success", map[string]interface{}{"dst_path": dstPath})
	return nil
}

func (m *Manager) CreateSymlink(ctx context.Context, target, linkPath string, user string) error {
	if err := m.validator.ValidatePath(linkPath); err != nil {
		m.logAudit(ctx, user, "create_symlink", linkPath, "failed", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("invalid link path: %w", err)
	}

	if err := os.Symlink(target, linkPath); err != nil {
		m.logAudit(ctx, user, "create_symlink", linkPath, "failed", map[string]interface{}{"error": err.Error(), "target": target})
		return fmt.Errorf("create symlink: %w", err)
	}

	m.logAudit(ctx, user, "create_symlink", linkPath, "success", map[string]interface{}{"target": target})
	return nil
}

func (m *Manager) CreateHardlink(ctx context.Context, target, linkPath string, user string) error {
	if err := m.validator.ValidatePath(target); err != nil {
		m.logAudit(ctx, user, "create_hardlink", linkPath, "failed", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("invalid target path: %w", err)
	}

	if err := m.validator.ValidatePath(linkPath); err != nil {
		m.logAudit(ctx, user, "create_hardlink", linkPath, "failed", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("invalid link path: %w", err)
	}

	if err := os.Link(target, linkPath); err != nil {
		m.logAudit(ctx, user, "create_hardlink", linkPath, "failed", map[string]interface{}{"error": err.Error(), "target": target})
		return fmt.Errorf("create hardlink: %w", err)
	}

	m.logAudit(ctx, user, "create_hardlink", linkPath, "success", map[string]interface{}{"target": target})
	return nil
}

func (m *Manager) buildFileInfo(path string, info os.FileInfo) FileInfo {
	fileInfo := FileInfo{
		Name:        info.Name(),
		Path:        path,
		Size:        info.Size(),
		Mode:        info.Mode(),
		ModTime:     info.ModTime(),
		IsDir:       info.IsDir(),
		IsSymlink:   info.Mode()&os.ModeSymlink != 0,
		Permissions: info.Mode().String(),
	}

	if owner, group, ok := getOwnerAndGroup(info); ok {
		fileInfo.Owner = owner
		fileInfo.Group = group
	}

	return fileInfo
}

func (m *Manager) logAudit(ctx context.Context, user, action, resource, result string, details map[string]interface{}) {
	if m.audit == nil {
		return
	}

	entry := &audit.Entry{
		Timestamp: time.Now(),
		User:      user,
		Action:    action,
		Resource:  resource,
		Result:    result,
		Details:   details,
	}

	m.audit.Log(ctx, entry)
}
