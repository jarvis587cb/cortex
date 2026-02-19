package main

import (
	"log"
	"net/http"
)

// methodAllowed wraps an HTTP handler to check if the request method is allowed
func methodAllowed(handler http.HandlerFunc, methods ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, method := range methods {
			if r.Method == method {
				handler(w, r)
				return
			}
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleError writes an error response
func handleError(w http.ResponseWriter, status int, message string, err error) {
	if err != nil {
		log.Printf("%s: %v", message, err)
	}
	http.Error(w, message, status)
}
