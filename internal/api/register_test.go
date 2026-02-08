package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func assertMuxPatterns(t *testing.T, mux *http.ServeMux, paths []string) {
	t.Helper()

	for _, path := range paths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		_, pattern := mux.Handler(req)
		if pattern != path {
			t.Fatalf("expected handler for %q, got pattern %q", path, pattern)
		}
	}
}

func TestAuthHandlersRegister(t *testing.T) {
	mux := http.NewServeMux()
	handler := &AuthHandlers{}
	handler.Register(mux)

	assertMuxPatterns(t, mux, []string{
		"/api/v1/auth/tokens/create",
		"/api/v1/auth/tokens",
		"/api/v1/auth/tokens/revoke",
		"/api/v1/auth/sessions/create",
		"/api/v1/auth/sessions/revoke",
	})
}

func TestDiskHandlersRegister(t *testing.T) {
	mux := http.NewServeMux()
	handler := &DiskHandlers{}
	handler.Register(mux)

	assertMuxPatterns(t, mux, []string{
		"/api/v1/disk/list",
		"/api/v1/disk/partitions",
		"/api/v1/disk/mount",
		"/api/v1/disk/unmount",
		"/api/v1/disk/smart",
	})
}

func TestIndexerHandlersRegister(t *testing.T) {
	mux := http.NewServeMux()
	handler := &IndexerHandlers{}
	handler.Register(mux)

	assertMuxPatterns(t, mux, []string{
		"/api/v1/indexer/scan",
		"/api/v1/indexer/search",
		"/api/v1/thumbnail/generate",
		"/api/v1/thumbnail/cleanup",
	})
}

func TestNetDiskHandlersRegister(t *testing.T) {
	mux := http.NewServeMux()
	handler := &NetDiskHandlers{}
	handler.Register(mux)

	assertMuxPatterns(t, mux, []string{
		"/api/v1/netdisk/shares",
		"/api/v1/netdisk/shares/add",
		"/api/v1/netdisk/shares/remove",
		"/api/v1/netdisk/mount",
		"/api/v1/netdisk/unmount",
		"/api/v1/netdisk/status",
	})
}

func TestNetManagerHandlersRegister(t *testing.T) {
	mux := http.NewServeMux()
	handler := &NetManagerHandlers{}
	handler.Register(mux)

	assertMuxPatterns(t, mux, []string{
		"/api/v1/network/interfaces",
		"/api/v1/network/interface",
		"/api/v1/network/config",
		"/api/v1/network/rollback",
		"/api/v1/network/history",
		"/api/v1/network/enable",
		"/api/v1/network/disable",
		"/api/v1/network/ports",
		"/api/v1/network/traffic",
	})
}

func TestSchedulerHandlersRegister(t *testing.T) {
	mux := http.NewServeMux()
	handler := &SchedulerHandlers{}
	handler.Register(mux)

	assertMuxPatterns(t, mux, []string{
		"/api/v1/scheduler/tasks",
		"/api/v1/scheduler/tasks/get",
		"/api/v1/scheduler/tasks/add",
		"/api/v1/scheduler/tasks/update",
		"/api/v1/scheduler/tasks/delete",
		"/api/v1/scheduler/tasks/execute",
		"/api/v1/scheduler/history",
	})
}

func TestShareHandlersRegister(t *testing.T) {
	mux := http.NewServeMux()
	handler := &ShareHandlers{}
	handler.Register(mux)

	assertMuxPatterns(t, mux, []string{
		"/api/v1/shares",
		"/api/v1/shares/get",
		"/api/v1/shares/add",
		"/api/v1/shares/update",
		"/api/v1/shares/remove",
		"/api/v1/shares/enable",
		"/api/v1/shares/disable",
		"/api/v1/shares/rollback",
	})
}
