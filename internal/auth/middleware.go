package auth

import (
	"log"
	"net/http"
	"strings"
)

// AuthMiddleware wraps an HTTP handler to require JWT authentication
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			log.Println("Missing Authorization header")
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		// Extract the token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			log.Println("Invalid Authorization header format")
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]

		// Validate the token
		claims, err := ValidateToken(tokenString)
		if err != nil {
			log.Printf("Token validation failed: %v", err)
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Log successful authentication
		log.Printf("Worker authenticated: %s from %s", claims.WorkerID, claims.URL)

		// Call the next handler
		next(w, r)
	}
}

// OptionalAuthMiddleware wraps a handler to allow but verify JWT if present
func OptionalAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString := parts[1]
				if claims, err := ValidateToken(tokenString); err == nil {
					log.Printf("Request authenticated: %s", claims.WorkerID)
				}
			}
		}

		// Call the next handler regardless
		next(w, r)
	}
}
