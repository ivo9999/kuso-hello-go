package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"hello":         "from kuso-hello-go",
			"database_url":  shorten(os.Getenv("DATABASE_URL"), 30),
			"redis_url":     shorten(os.Getenv("REDIS_URL"), 30),
			"path":          r.URL.Path,
			"host":          r.Host,
			"build":         os.Getenv("KUSO_BUILD_REF"),
		})
	})
	addr := ":" + port
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func shorten(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
