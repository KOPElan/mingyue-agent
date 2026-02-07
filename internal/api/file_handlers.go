package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/KOPElan/mingyue-agent/internal/audit"
	"github.com/KOPElan/mingyue-agent/internal/filemanager"
)

type FileAPI struct {
	manager *filemanager.Manager
	audit   *audit.Logger
}

func NewFileAPI(manager *filemanager.Manager, auditLogger *audit.Logger) *FileAPI {
	return &FileAPI{
		manager: manager,
		audit:   auditLogger,
	}
}

func (api *FileAPI) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/files/list", api.handleList)
	mux.HandleFunc("/api/v1/files/info", api.handleInfo)
	mux.HandleFunc("/api/v1/files/mkdir", api.handleMkdir)
	mux.HandleFunc("/api/v1/files/delete", api.handleDelete)
	mux.HandleFunc("/api/v1/files/rename", api.handleRename)
	mux.HandleFunc("/api/v1/files/copy", api.handleCopy)
	mux.HandleFunc("/api/v1/files/move", api.handleMove)
	mux.HandleFunc("/api/v1/files/upload", api.handleUpload)
	mux.HandleFunc("/api/v1/files/download", api.handleDownload)
	mux.HandleFunc("/api/v1/files/symlink", api.handleSymlink)
	mux.HandleFunc("/api/v1/files/hardlink", api.handleHardlink)
	mux.HandleFunc("/api/v1/files/checksum", api.handleChecksum)
}

func (api *FileAPI) handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "method not allowed"})
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Error: "path required"})
		return
	}

	opts := filemanager.ListOptions{
		Path: path,
	}

	user := getUser(r)
	files, err := api.manager.List(r.Context(), opts, user)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: files})
}

func (api *FileAPI) handleInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "method not allowed"})
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Error: "path required"})
		return
	}

	user := getUser(r)
	info, err := api.manager.GetInfo(r.Context(), path, user)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: info})
}

func (api *FileAPI) handleMkdir(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "method not allowed"})
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Error: "invalid request"})
		return
	}

	user := getUser(r)
	if err := api.manager.CreateDir(r.Context(), req.Path, user); err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true})
}

func (api *FileAPI) handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "method not allowed"})
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Error: "invalid request"})
		return
	}

	user := getUser(r)
	if err := api.manager.Delete(r.Context(), req.Path, user); err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true})
}

func (api *FileAPI) handleRename(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "method not allowed"})
		return
	}

	var req struct {
		OldPath string `json:"old_path"`
		NewPath string `json:"new_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Error: "invalid request"})
		return
	}

	user := getUser(r)
	if err := api.manager.Rename(r.Context(), req.OldPath, req.NewPath, user); err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true})
}

func (api *FileAPI) handleCopy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "method not allowed"})
		return
	}

	var req struct {
		SrcPath string `json:"src_path"`
		DstPath string `json:"dst_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Error: "invalid request"})
		return
	}

	user := getUser(r)
	if err := api.manager.Copy(r.Context(), req.SrcPath, req.DstPath, user); err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true})
}

func (api *FileAPI) handleMove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "method not allowed"})
		return
	}

	var req struct {
		SrcPath string `json:"src_path"`
		DstPath string `json:"dst_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Error: "invalid request"})
		return
	}

	user := getUser(r)
	if err := api.manager.Move(r.Context(), req.SrcPath, req.DstPath, user); err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true})
}

func (api *FileAPI) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "method not allowed"})
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Error: "path required"})
		return
	}

	maxSize := int64(10 * 1024 * 1024 * 1024)
	if maxSizeStr := r.URL.Query().Get("max_size"); maxSizeStr != "" {
		if size, err := strconv.ParseInt(maxSizeStr, 10, 64); err == nil {
			maxSize = size
		}
	}

	opts := filemanager.UploadOptions{
		Path:    path,
		MaxSize: maxSize,
	}

	user := getUser(r)
	if err := api.manager.Upload(r.Context(), r.Body, opts, user); err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true})
}

func (api *FileAPI) handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "method not allowed"})
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Error: "path required"})
		return
	}

	info, err := api.manager.GetInfo(r.Context(), path, getUser(r))
	if err != nil {
		writeJSON(w, http.StatusNotFound, Response{Success: false, Error: "file not found"})
		return
	}

	opts := filemanager.DownloadOptions{
		Path: path,
	}

	rangeHeader := r.Header.Get("Range")
	if rangeHeader != "" {
		start, end, err := filemanager.ParseRangeHeader(rangeHeader, info.Size)
		if err == nil {
			opts.RangeStart = start
			opts.RangeEnd = end
			w.Header().Set("Content-Range", "bytes "+strconv.FormatInt(start, 10)+"-"+strconv.FormatInt(end, 10)+"/"+strconv.FormatInt(info.Size, 10))
			w.WriteHeader(http.StatusPartialContent)
		}
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+info.Name+"\"")

	user := getUser(r)
	if _, err := api.manager.Download(r.Context(), w, opts, user); err != nil {
		return
	}
}

func (api *FileAPI) handleSymlink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "method not allowed"})
		return
	}

	var req struct {
		Target   string `json:"target"`
		LinkPath string `json:"link_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Error: "invalid request"})
		return
	}

	user := getUser(r)
	if err := api.manager.CreateSymlink(r.Context(), req.Target, req.LinkPath, user); err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true})
}

func (api *FileAPI) handleHardlink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "method not allowed"})
		return
	}

	var req struct {
		Target   string `json:"target"`
		LinkPath string `json:"link_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Error: "invalid request"})
		return
	}

	user := getUser(r)
	if err := api.manager.CreateHardlink(r.Context(), req.Target, req.LinkPath, user); err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true})
}

func (api *FileAPI) handleChecksum(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "method not allowed"})
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Error: "path required"})
		return
	}

	user := getUser(r)
	checksum, err := api.manager.GetChecksum(r.Context(), path, user)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: map[string]string{"checksum": checksum}})
}

func getUser(r *http.Request) string {
	user := r.Header.Get("X-User")
	if user == "" {
		user = "anonymous"
	}
	return user
}
