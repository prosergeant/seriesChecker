package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/prosergeant/seriesChecker/internal/service"
)

const UserIDKey = "user_id"

func Auth(sessionService *service.SessionService, cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(cookieName)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized"}`))
				return
			}

			sessionID, err := uuid.Parse(cookie.Value)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized"}`))
				return
			}

			userID, err := sessionService.Get(r.Context(), sessionID)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized"}`))
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID.String())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type contextKey string

func WithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, contextKey(UserIDKey), userID)
}

func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	userIDStr, ok := ctx.Value(UserIDKey).(string)

	if !ok {
		return uuid.Nil, ok
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, ok

	}

	return userID, ok
}
