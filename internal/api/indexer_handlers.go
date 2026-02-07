package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/KOPElan/mingyue-agent/internal/audit"
	"github.com/KOPElan/mingyue-agent/internal/indexer"
	"github.com/KOPElan/mingyue-agent/internal/thumbnail"
)

type IndexerHandlers struct {
	indexer   *indexer.Indexer
	thumbnail *thumbnail.Generator
	audit     *audit.Logger
}

func NewIndexerHandlers(idx *indexer.Indexer, thumb *thumbnail.Generator, auditLogger *audit.Logger) *IndexerHandlers {
	return &IndexerHandlers{
		indexer:   idx,
		thumbnail: thumb,
		audit:     auditLogger,
	}
}

// ScanFiles godoc
// @Summary Scan files for indexing
// @Description Scans specified paths and indexes file metadata
// @Tags indexer
// @Accept json
// @Produce json
// @Param body body indexer.ScanOptions true "Scan options"
// @Success 200 {object} Response{data=indexer.ScanResult}
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /indexer/scan [post]
// @Security UserAuth
func (h *IndexerHandlers) ScanFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "method not allowed"})
		return
	}

	var opts indexer.ScanOptions
	if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Error: "invalid request body"})
		return
	}

	result, err := h.indexer.Scan(r.Context(), opts)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), &audit.Entry{
			User:     getUser(r),
			Action:   "scan_files",
			Resource: "indexer",
			Result:   "success",
			SourceIP: r.RemoteAddr,
			Details:  map[string]interface{}{"paths": opts.Paths, "files_scanned": result.FilesScanned},
		})
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: result})
}

// SearchFiles godoc
// @Summary Search indexed files
// @Description Searches indexed files by query string
// @Tags indexer
// @Produce json
// @Param q query string true "Search query"
// @Param limit query int false "Result limit" default(50)
// @Param offset query int false "Result offset" default(0)
// @Success 200 {object} Response{data=[]indexer.FileMetadata}
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /indexer/search [get]
func (h *IndexerHandlers) SearchFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "method not allowed"})
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Error: "query parameter required"})
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	results, err := h.indexer.Search(r.Context(), query, limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: results})
}

// GenerateThumbnail godoc
// @Summary Generate thumbnail for file
// @Description Generates a thumbnail for the specified file
// @Tags thumbnail
// @Produce json
// @Param path query string true "File path"
// @Success 200 {object} Response{data=thumbnail.ThumbnailInfo}
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /thumbnail/generate [post]
// @Security UserAuth
func (h *IndexerHandlers) GenerateThumbnail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "method not allowed"})
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Error: "path parameter required"})
		return
	}

	thumbInfo, err := h.thumbnail.Generate(r.Context(), path)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	// Update indexer with thumbnail URL
	if err := h.indexer.UpdateThumbnailURL(path, thumbInfo.ThumbPath); err != nil {
		// Non-fatal, log but continue
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), &audit.Entry{
			User:     getUser(r),
			Action:   "generate_thumbnail",
			Resource: path,
			Result:   "success",
			SourceIP: r.RemoteAddr,
		})
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: thumbInfo})
}

// CleanupCache godoc
// @Summary Cleanup thumbnail cache
// @Description Removes old or excess thumbnails from cache
// @Tags thumbnail
// @Produce json
// @Success 200 {object} Response
// @Failure 500 {object} Response
// @Router /thumbnail/cleanup [post]
// @Security UserAuth
func (h *IndexerHandlers) CleanupCache(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "method not allowed"})
		return
	}

	if err := h.thumbnail.Cleanup(context.Background()); err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), &audit.Entry{
			User:     getUser(r),
			Action:   "cleanup_cache",
			Resource: "thumbnail",
			Result:   "success",
			SourceIP: r.RemoteAddr,
		})
	}

	writeJSON(w, http.StatusOK, Response{Success: true})
}

func getUser(r *http.Request) string {
	if user := r.Header.Get("X-User"); user != "" {
		return user
	}
	return "unknown"
}
