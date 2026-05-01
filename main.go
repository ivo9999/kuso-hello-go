// Tiny HTTP server to smoke-test kuso end-to-end.
//
// GET /          -> JSON greeting + a few env vars (echoes secret presence
//                   without leaking values)
// GET /env       -> full env (filtered: KUSO_*, DATABASE_URL, REDIS_URL,
//                   anything starting with PUBLIC_, plus any names the
//                   caller asked for via ?name=)
// GET /secret    -> proves a secret env var was injected (returns
//                   true/false, not the value)
// GET /error     -> returns 500 with a JSON message; useful for testing
//                   how kuso surfaces app errors in logs
// GET /panic     -> intentionally panics; tests that a crash → CrashLoop
//                   shows up in kuso's UI
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", root)
	mux.HandleFunc("/env", showEnv)
	mux.HandleFunc("/secret", showSecret)
	mux.HandleFunc("/error", returnError)
	mux.HandleFunc("/panic", causePanic)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Printf("kuso-hello-go listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, logging(mux)))
}

func root(w http.ResponseWriter, r *http.Request) {
	jsonOut(w, http.StatusOK, map[string]any{
		"hello":               "from kuso-hello-go",
		"path":                r.URL.Path,
		"host":                r.Host,
		"build_ref":           os.Getenv("KUSO_BUILD_REF"),
		"has_database_url":    os.Getenv("DATABASE_URL") != "",
		"has_redis_url":       os.Getenv("REDIS_URL") != "",
		"has_test_secret":     os.Getenv("TEST_SECRET") != "",
		"has_log_level":       os.Getenv("LOG_LEVEL") != "",
		"server_time":         time.Now().UTC().Format(time.RFC3339),
	})
}

// /env returns env vars with a deny-all-by-default filter. Plus user-
// specified names via ?name=FOO&name=BAR. Never returns names containing
// PASSWORD/TOKEN/SECRET unless explicitly requested via ?name=.
func showEnv(w http.ResponseWriter, r *http.Request) {
	names := r.URL.Query()["name"]
	allowed := map[string]bool{}
	for _, n := range names {
		allowed[n] = true
	}
	out := map[string]string{}
	for _, kv := range os.Environ() {
		i := strings.IndexByte(kv, '=')
		if i <= 0 {
			continue
		}
		k, v := kv[:i], kv[i+1:]
		if allowed[k] {
			out[k] = v
			continue
		}
		// Default safe list.
		switch {
		case strings.HasPrefix(k, "KUSO_"):
			out[k] = v
		case strings.HasPrefix(k, "PUBLIC_"):
			out[k] = v
		case k == "PORT", k == "HOSTNAME", k == "LOG_LEVEL":
			out[k] = v
		}
	}
	jsonOut(w, http.StatusOK, out)
}

// /secret returns whether a particular secret env var is present and a
// length/checksum-style fingerprint, never the value. Useful for
// validating that `kuso secret set` actually injected something.
func showSecret(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "TEST_SECRET"
	}
	v := os.Getenv(name)
	out := map[string]any{
		"name":    name,
		"present": v != "",
		"length":  len(v),
	}
	if v != "" {
		// Tiny non-cryptographic fingerprint so the user can verify a
		// rotation without seeing the value. djb2.
		var h uint32 = 5381
		for i := 0; i < len(v); i++ {
			h = ((h << 5) + h) + uint32(v[i])
		}
		out["fingerprint"] = fmt.Sprintf("%08x", h)
	}
	jsonOut(w, http.StatusOK, out)
}

// /error always returns 500. Use to verify kuso shows app errors in
// logs / metrics.
func returnError(w http.ResponseWriter, r *http.Request) {
	msg := r.URL.Query().Get("msg")
	if msg == "" {
		msg = "intentional error from kuso-hello-go /error"
	}
	jsonOut(w, http.StatusInternalServerError, map[string]any{
		"error":   msg,
		"path":    r.URL.Path,
		"trigger": "intentional",
	})
}

// /panic crashes the process. The orchestrator restarts the pod; kuso's
// "Recent crashes" view should pick this up.
func causePanic(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = w.Write([]byte("panic incoming\n"))
	panic("intentional panic from kuso-hello-go /panic")
}

func jsonOut(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func logging(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		h.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
