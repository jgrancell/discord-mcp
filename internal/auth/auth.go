// Package auth is the inbound gate for the serve HTTP transport:
// loopback-only by default, a static bearer challenge otherwise.
package auth

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// bearer extracts the token from an Authorization header, or "".
func bearer(header string) string {
	fields := strings.Fields(header)
	if len(fields) != 2 || !strings.EqualFold(fields[0], "bearer") {
		return ""
	}
	return fields[1]
}

// Middleware guards next with a static bearer token: requests without a
// matching Authorization header are rejected with 401 and a
// WWW-Authenticate challenge. There is no authorization server to point
// at — the credential is a single configured value, compared in constant
// time.
func Middleware(next http.Handler, token string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := bearer(r.Header.Get("Authorization"))
		if got == "" || subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="discord-mcp"`)
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
