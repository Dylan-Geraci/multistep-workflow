package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/dylangeraci/flowforge/internal/config"
	"github.com/dylangeraci/flowforge/internal/middleware"
	"github.com/dylangeraci/flowforge/internal/model"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	db  *pgxpool.Pool
	cfg config.Config
}

func NewAuthHandler(db *pgxpool.Pool, cfg config.Config) *AuthHandler {
	return &AuthHandler{db: db, cfg: cfg}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.WriteError(w, http.StatusBadRequest, "INVALID_JSON", "Request body is not valid JSON")
		return
	}

	if !strings.Contains(req.Email, "@") {
		model.WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "Invalid email address")
		return
	}
	if len(req.Password) < 8 {
		model.WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "Password must be at least 8 characters")
		return
	}
	if req.DisplayName == "" {
		model.WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "display_name is required")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		model.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Failed to hash password")
		return
	}

	userID := newULID()
	now := time.Now().UTC()

	_, err = h.db.Exec(r.Context(),
		`INSERT INTO users (id, email, password_hash, display_name, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, req.Email, string(hash), req.DisplayName, now, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			model.WriteError(w, http.StatusConflict, "EMAIL_EXISTS", "A user with this email already exists")
			return
		}
		model.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Failed to create user")
		return
	}

	resp, err := h.issueTokenPair(r.Context(), userID)
	if err != nil {
		model.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Failed to issue tokens")
		return
	}

	model.WriteJSON(w, http.StatusCreated, resp)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.WriteError(w, http.StatusBadRequest, "INVALID_JSON", "Request body is not valid JSON")
		return
	}

	var userID, passwordHash string
	err := h.db.QueryRow(r.Context(),
		`SELECT id, password_hash FROM users WHERE email = $1`, req.Email,
	).Scan(&userID, &passwordHash)
	if err != nil {
		model.WriteError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		model.WriteError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
		return
	}

	resp, err := h.issueTokenPair(r.Context(), userID)
	if err != nil {
		model.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Failed to issue tokens")
		return
	}

	model.WriteJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req model.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.WriteError(w, http.StatusBadRequest, "INVALID_JSON", "Request body is not valid JSON")
		return
	}

	if req.RefreshToken == "" {
		model.WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "refresh_token is required")
		return
	}

	tokenHash := sha256Hex(req.RefreshToken)

	var tokenID, userID string
	var expiresAt time.Time
	var revokedAt *time.Time
	err := h.db.QueryRow(r.Context(),
		`SELECT id, user_id, expires_at, revoked_at FROM refresh_tokens WHERE token_hash = $1`,
		tokenHash,
	).Scan(&tokenID, &userID, &expiresAt, &revokedAt)
	if err != nil {
		model.WriteError(w, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "Invalid refresh token")
		return
	}

	if revokedAt != nil {
		model.WriteError(w, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "Refresh token has been revoked")
		return
	}
	if time.Now().After(expiresAt) {
		model.WriteError(w, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "Refresh token has expired")
		return
	}

	// Revoke old token
	_, err = h.db.Exec(r.Context(),
		`UPDATE refresh_tokens SET revoked_at = now() WHERE id = $1`, tokenID,
	)
	if err != nil {
		model.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Failed to revoke old token")
		return
	}

	resp, err := h.issueTokenPair(r.Context(), userID)
	if err != nil {
		model.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Failed to issue tokens")
		return
	}

	model.WriteJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req model.LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if req.RefreshToken != "" {
		tokenHash := sha256Hex(req.RefreshToken)
		h.db.Exec(r.Context(),
			`UPDATE refresh_tokens SET revoked_at = now() WHERE token_hash = $1 AND revoked_at IS NULL`,
			tokenHash,
		)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	var user model.UserResponse
	err := h.db.QueryRow(r.Context(),
		`SELECT id, email, display_name, created_at, updated_at FROM users WHERE id = $1`,
		userID,
	).Scan(&user.ID, &user.Email, &user.DisplayName, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			model.WriteError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
			return
		}
		model.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Failed to fetch user")
		return
	}

	model.WriteJSON(w, http.StatusOK, user)
}

func (h *AuthHandler) issueTokenPair(ctx context.Context, userID string) (*model.AuthResponse, error) {
	now := time.Now()
	exp := now.Add(h.cfg.AccessTokenTTL)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"iat": now.Unix(),
		"exp": exp.Unix(),
	})
	accessToken, err := token.SignedString([]byte(h.cfg.JWTSecret))
	if err != nil {
		return nil, err
	}

	// Generate opaque refresh token
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return nil, err
	}
	refreshToken := hex.EncodeToString(rawBytes)
	tokenHash := sha256Hex(refreshToken)

	tokenID := newULID()
	expiresAt := now.Add(h.cfg.RefreshTokenTTL)

	_, err = h.db.Exec(ctx,
		`INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, now())`,
		tokenID, userID, tokenHash, expiresAt,
	)
	if err != nil {
		return nil, err
	}

	return &model.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(h.cfg.AccessTokenTTL.Seconds()),
	}, nil
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
