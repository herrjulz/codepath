// Package server is the loopback HTTP API and the embedded walkthrough UI.
package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"codepath/internal/mcp"
	"codepath/internal/reports"
	"codepath/internal/service"
	"codepath/internal/store"
)

// Handler serves the API, Streamable HTTP MCP at /mcp, and when assets is
// non-nil, the single-page UI.
func Handler(svc *service.Service, token string, assets http.FileSystem) http.Handler {
	h := &handlers{svc: svc, token: token}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /api/reports", h.list)
	mux.HandleFunc("POST /api/reports", h.create)
	mux.HandleFunc("GET /api/reports/{id}", h.get)
	mux.HandleFunc("GET /api/reports/{id}/source", h.source)
	mcpHandler := loopbackOnly(mcp.HTTPHandler(svc))
	mux.Handle("/mcp", mcpHandler)
	mux.Handle("/mcp/", mcpHandler)
	if assets != nil {
		mux.Handle("GET /assets/", assetsHandler(assets))
		mux.HandleFunc("GET /reports/{id}", func(w http.ResponseWriter, r *http.Request) {
			if !allowRequest(w, r) {
				return
			}
			serveIndex(w, r, assets)
		})
		mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
			if !allowRequest(w, r) {
				return
			}
			serveIndex(w, r, assets)
		})
	}
	return securityHeaders(mux)
}

func loopbackOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !allowRequest(w, r) {
			return
		}
		next.ServeHTTP(w, r)
	})
}

type handlers struct {
	svc   *service.Service
	token string
}

func (h *handlers) health(w http.ResponseWriter, r *http.Request) {
	if !allowRequest(w, r) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"repo":   repoName(h.svc.RepoRoot),
	})
}

func (h *handlers) list(w http.ResponseWriter, r *http.Request) {
	if !allowRequest(w, r) {
		return
	}
	items, err := h.svc.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "Could not list reports.", nil)
		return
	}
	if items == nil {
		items = []reports.Summary{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *handlers) get(w http.ResponseWriter, r *http.Request) {
	if !allowRequest(w, r) {
		return
	}
	rep, err := h.svc.Get(r.PathValue("id"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

func (h *handlers) source(w http.ResponseWriter, r *http.Request) {
	if !allowRequest(w, r) {
		return
	}
	refIndex := 0
	if raw := r.URL.Query().Get("refIndex"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "refIndex must be an integer.", nil)
			return
		}
		refIndex = n
	}
	view, err := h.svc.Source(r.PathValue("id"), r.URL.Query().Get("stepId"), refIndex)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, view)
}

func (h *handlers) create(w http.ResponseWriter, r *http.Request) {
	if !allowRequest(w, r) {
		return
	}
	if !authorized(r, h.token) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "A local bearer token is required.", nil)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, reports.MaxBodyBytes))
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "too_large", "Request body exceeds 1 MiB.", nil)
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", "Could not read the request body.", nil)
		return
	}
	result, err := h.svc.Create(r.Context(), body)
	if err != nil {
		writeCreateError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func writeStoreError(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "Report or code reference not found.", nil)
		return
	}
	writeError(w, http.StatusInternalServerError, "internal", "Could not read the report.", nil)
}

func writeCreateError(w http.ResponseWriter, err error) {
	if errors.Is(err, service.ErrTooLarge) {
		writeError(w, http.StatusRequestEntityTooLarge, "too_large", "Request body exceeds 1 MiB.", nil)
		return
	}
	var ve *reports.ValidationError
	if errors.As(err, &ve) {
		writeError(w, http.StatusBadRequest, "validation_failed", "The walkthrough failed validation.", ve.Details)
		return
	}
	writeError(w, http.StatusBadRequest, "bad_request", err.Error(), nil)
}

type errorBody struct {
	Error   string               `json:"error"`
	Message string               `json:"message"`
	Details []reports.FieldError `json:"details,omitempty"`
}

func writeError(w http.ResponseWriter, status int, code, message string, details []reports.FieldError) {
	writeJSON(w, status, errorBody{Error: code, Message: message, Details: details})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if w.Header().Get("Cache-Control") == "" {
		w.Header().Set("Cache-Control", "no-store")
	}
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	_ = enc.Encode(value)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self' data:; connect-src 'self'; worker-src 'self' blob:; child-src 'self' blob:; frame-ancestors 'none'; base-uri 'self'; form-action 'none'")
		next.ServeHTTP(w, r)
	})
}

func repoName(root string) string {
	name := root
	if i := strings.LastIndex(root, "/"); i >= 0 {
		name = root[i+1:]
	}
	return name
}
