package handlers

import (
	"net/http"

	"research-data-collection/internal/config"
)

func BasicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		c := config.Get()
		if !ok || user != c.AdminUser || pass != c.AdminPass {
			w.Header().Set("WWW-Authenticate", `Basic realm="Dashboard"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}
