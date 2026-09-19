package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"mmadle/backend/internal/store"
)

// Account listing + activation switch for the admin users view. Same gating
// as the metrics interface: guests 401, players 404, admins only.

// adminUsersPage parses the listing query: ?q= (partial username, ≤64
// chars), ?limit= (default 20, 1..100), ?offset= (default 0, ≥0).
func adminUsersPage(w http.ResponseWriter, r *http.Request) (q string, limit, offset int, ok bool) {
	q = strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) > 64 {
		writeError(w, http.StatusBadRequest, "invalid_query", "q must be at most 64 characters")
		return "", 0, 0, false
	}
	limit = 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100 {
			writeError(w, http.StatusBadRequest, "invalid_limit", "limit must be an integer between 1 and 100")
			return "", 0, 0, false
		}
		limit = n
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("offset")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			writeError(w, http.StatusBadRequest, "invalid_offset", "offset must be a non-negative integer")
			return "", 0, 0, false
		}
		offset = n
	}
	return q, limit, offset, true
}

// GET /api/admin/users — paginated account listing with status and lifetime
// game totals, plus the filtered totals for the numbers row.
func (s *Server) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	if s.AdminMetrics == nil {
		writeError(w, http.StatusInternalServerError, "admin_unavailable", "admin interface is not configured")
		return
	}
	q, limit, offset, ok := adminUsersPage(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	page, err := s.AdminMetrics.ListUsers(ctx, q, limit, offset)
	if err != nil {
		s.Logger.Error("admin users list failed")
		writeError(w, http.StatusInternalServerError, "users_failed", "could not load accounts")
		return
	}
	writeJSON(w, http.StatusOK, page)
}

type setActiveRequest struct {
	UserID int64 `json:"user_id"`
	Active bool  `json:"active"`
}

// POST /api/admin/users/active — flip one account's switch. Deactivation
// drops every session (immediate lockout, enforced again at session
// resolution). Self-deactivation is refused: nothing could re-enable the
// account afterwards.
func (s *Server) handleAdminSetActive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	if !requireJSONContentType(w, r) {
		return
	}
	admin, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	if s.AdminMetrics == nil {
		writeError(w, http.StatusInternalServerError, "admin_unavailable", "admin interface is not configured")
		return
	}
	var req setActiveRequest
	if !decodeStrict(w, r, &req) {
		return
	}
	if req.UserID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_user_id", "user_id must be a positive integer")
		return
	}
	if req.UserID == admin.ID {
		writeError(w, http.StatusBadRequest, "cannot_change_self", "an admin cannot deactivate their own account")
		return
	}
	// SetUserActive lives on the auth store (sessions die with the account).
	if s.Auth == nil {
		writeError(w, http.StatusInternalServerError, "admin_unavailable", "admin interface is not configured")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	if err := s.Auth.SetUserActive(ctx, req.UserID, req.Active); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "user_not_found", "unknown account id")
			return
		}
		s.Logger.Error("admin set-active failed", "user_id", req.UserID)
		writeError(w, http.StatusInternalServerError, "users_failed", "could not update account")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user_id": req.UserID, "active": req.Active})
}
