package internal

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
)

func BuildMux(cfg *Config, st *FileStore, mgr *Manager) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	})
	mux.HandleFunc("/apps", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, mgr.Apps())
	})
	mux.HandleFunc("/poll", func(w http.ResponseWriter, r *http.Request) {
		appID := r.URL.Query().Get("appId")
		country := r.URL.Query().Get("country")
		if appID == "" || country == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "appId and country are required"})
			return
		}
		go mgr.PollOnce(context.Background(), AppConfig{AppID: appID, Country: country})
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "poll started"})
	})
	mux.HandleFunc("/reviews", func(w http.ResponseWriter, r *http.Request) {
		appID := r.URL.Query().Get("appId")
		country := r.URL.Query().Get("country")
		if appID == "" || country == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "appId and country are required"})
			return
		}
		hours := 48
		if hs := r.URL.Query().Get("hours"); hs != "" {
			if n, err := strconv.Atoi(hs); err == nil && n > 0 && n <= 24*90 {
				hours = n
			}
		}
		revs, err := st.ReadRecent(appID, country, time.Duration(hours)*time.Hour)
		if err != nil {
			log.Printf("read recent error: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		now := time.Now().UTC()
		resp := map[string]any{
			"appId": appID, "country": country,
			"from":    now.Add(-time.Duration(hours) * time.Hour).Format(time.RFC3339),
			"to":      now.Format(time.RFC3339),
			"count":   len(revs),
			"reviews": revs,
		}
		writeJSON(w, http.StatusOK, resp)
	})
	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func WithCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
