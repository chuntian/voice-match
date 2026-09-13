package report

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

type ctxKey string

const userIDCtxKey ctxKey = "report_user_id"

// TokenValidator 由外部（user.Service）提供
type TokenValidator interface {
	ValidateToken(token string) (string, error)
}

// Handler 举报模块 HTTP handler
type Handler struct {
	svc      *Service
	auth     TokenValidator
	adminIDs map[string]struct{}
}

// NewHandler 构造 handler
func NewHandler(svc *Service, tv TokenValidator, adminIDs []string) *Handler {
	m := make(map[string]struct{}, len(adminIDs))
	for _, id := range adminIDs {
		if id != "" {
			m[id] = struct{}{}
		}
	}
	return &Handler{svc: svc, auth: tv, adminIDs: m}
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	authed := h.AuthMiddleware

	mux.Handle("POST /api/v1/reports", authed(http.HandlerFunc(h.submitReport)))
	mux.Handle("GET /api/v1/reports", h.adminChain(http.HandlerFunc(h.listReports)))
	mux.Handle("PUT /api/v1/reports/{id}/status", h.adminChain(http.HandlerFunc(h.updateStatus)))

	mux.Handle("POST /api/v1/users/me/blacklist", authed(http.HandlerFunc(h.blockUser)))
	mux.Handle("DELETE /api/v1/users/me/blacklist/{blockedUserID}", authed(http.HandlerFunc(h.unblockUser)))
	mux.Handle("GET /api/v1/users/me/blacklist", authed(http.HandlerFunc(h.listBlacklist)))
}

// ---------- middleware ----------

// AuthMiddleware Bearer token -> userID
func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			writeErr(w, http.StatusUnauthorized, "missing authorization header")
			return
		}
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeErr(w, http.StatusUnauthorized, "invalid authorization scheme")
			return
		}
		uid, err := h.auth.ValidateToken(parts[1])
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "invalid token")
			return
		}
		ctx := context.WithValue(r.Context(), userIDCtxKey, uid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AdminMiddleware 校验是否管理员（需先经过 AuthMiddleware）
func (h *Handler) AdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, _ := r.Context().Value(userIDCtxKey).(string)
		if _, ok := h.adminIDs[uid]; !ok {
			writeErr(w, http.StatusForbidden, "admin only")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// adminChain = AuthMiddleware -> AdminMiddleware -> handler
func (h *Handler) adminChain(next http.Handler) http.Handler {
	return h.AuthMiddleware(h.AdminMiddleware(next))
}

func userIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(userIDCtxKey).(string)
	return v
}

// ---------- handlers ----------

func (h *Handler) submitReport(w http.ResponseWriter, r *http.Request) {
	uid := userIDFromCtx(r.Context())
	var req SubmitReportRequest
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	rep, err := h.svc.SubmitReport(r.Context(), uid, req.ReportedID, req.Reason, req.Content)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, rep)
}

func (h *Handler) listReports(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	status := q.Get("status")
	res, err := h.svc.ListReports(r.Context(), status, page, pageSize)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *Handler) updateStatus(w http.ResponseWriter, r *http.Request) {
	reportID := r.PathValue("id")
	var req UpdateStatusRequest
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.ProcessReport(r.Context(), reportID, req.Status); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": req.Status})
}

func (h *Handler) blockUser(w http.ResponseWriter, r *http.Request) {
	uid := userIDFromCtx(r.Context())
	var req BlockRequest
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.BlockUser(r.Context(), uid, req.BlockedUserID); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "blocked"})
}

func (h *Handler) unblockUser(w http.ResponseWriter, r *http.Request) {
	uid := userIDFromCtx(r.Context())
	blocked := r.PathValue("blockedUserID")
	if blocked == "" {
		writeErr(w, http.StatusBadRequest, "missing blocked user id")
		return
	}
	if err := h.svc.UnblockUser(r.Context(), uid, blocked); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "unblocked"})
}

func (h *Handler) listBlacklist(w http.ResponseWriter, r *http.Request) {
	uid := userIDFromCtx(r.Context())
	list, err := h.svc.GetBlacklist(r.Context(), uid)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []BlockedUser{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": list})
}

// ---------- helpers ----------

var errEmptyBody = errors.New("empty body")

func decodeJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	if r.Body == nil {
		return errEmptyBody
	}
	dec := json.NewDecoder(r.Body)
	return dec.Decode(v)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
