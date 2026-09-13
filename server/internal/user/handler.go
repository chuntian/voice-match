package user

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type ctxKey string

const userIDCtxKey ctxKey = "user_id"

// Handler 用户模块 HTTP handler
type Handler struct {
	svc *Service
}

// NewHandler 构造 handler
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes 注册路由到标准库 ServeMux（Go 1.22+ method 模式）
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/apple", h.appleLogin)
	mux.HandleFunc("POST /api/v1/auth/phone", h.sendPhoneCode)
	mux.HandleFunc("POST /api/v1/auth/phone/verify", h.phoneLogin)

	auth := h.AuthMiddleware
	mux.Handle("GET /api/v1/users/me", auth(http.HandlerFunc(h.getMe)))
	mux.Handle("PUT /api/v1/users/me", auth(http.HandlerFunc(h.updateMe)))
	mux.Handle("GET /api/v1/users/me/preferences", auth(http.HandlerFunc(h.getPrefs)))
	mux.Handle("PUT /api/v1/users/me/preferences", auth(http.HandlerFunc(h.putPrefs)))
}

// ---------- middleware ----------

// AuthMiddleware 从 Authorization: Bearer <token> 解析用户 ID 注入 context
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
		userID, err := h.svc.ValidateToken(parts[1])
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "invalid token: "+err.Error())
			return
		}
		ctx := context.WithValue(r.Context(), userIDCtxKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserIDFromContext 从 context 取用户 ID
func UserIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userIDCtxKey).(string)
	return v, ok && v != ""
}

// ---------- handlers ----------

func (h *Handler) appleLogin(w http.ResponseWriter, r *http.Request) {
	var req AppleLoginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	u, err := h.svc.LoginWithApple(r.Context(), req.IdentityToken)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "apple login failed: "+err.Error())
		return
	}
	token, err := h.svc.GenerateToken(u.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gen token: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, LoginResponse{Token: token, User: u})
}

func (h *Handler) sendPhoneCode(w http.ResponseWriter, r *http.Request) {
	var req SendCodeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.SendPhoneCode(r.Context(), req.Phone); err != nil {
		writeErr(w, http.StatusInternalServerError, "send code: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (h *Handler) phoneLogin(w http.ResponseWriter, r *http.Request) {
	var req VerifyCodeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	u, err := h.svc.LoginWithPhone(r.Context(), req.Phone, req.Code)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "phone login failed: "+err.Error())
		return
	}
	token, err := h.svc.GenerateToken(u.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gen token: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, LoginResponse{Token: token, User: u})
}

func (h *Handler) getMe(w http.ResponseWriter, r *http.Request) {
	uid, _ := UserIDFromContext(r.Context())
	u, err := h.svc.GetProfile(r.Context(), uid)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeErr(w, http.StatusNotFound, "user not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (h *Handler) updateMe(w http.ResponseWriter, r *http.Request) {
	uid, _ := UserIDFromContext(r.Context())
	var req UpdateProfileRequest
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	u, err := h.svc.UpdateProfile(r.Context(), uid, req)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (h *Handler) getPrefs(w http.ResponseWriter, r *http.Request) {
	uid, _ := UserIDFromContext(r.Context())
	p, err := h.svc.GetPreferences(r.Context(), uid)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) putPrefs(w http.ResponseWriter, r *http.Request) {
	uid, _ := UserIDFromContext(r.Context())
	var p Preferences
	if err := decodeJSON(r, &p); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	updated, err := h.svc.UpdatePreferences(r.Context(), uid, &p)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// ---------- helpers ----------

func decodeJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
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
