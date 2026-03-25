package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"ai-rss-reader/internal/ai"
	"ai-rss-reader/internal/config"
	"ai-rss-reader/internal/events"
	"ai-rss-reader/internal/models"
	"ai-rss-reader/internal/opml"
	"ai-rss-reader/internal/repository/sqlite"
	"ai-rss-reader/internal/service"
)

// Global services (same as main.go)
var (
	rssService     *service.RSSService
	filterService  *service.FilterService
	summaryService *service.SummaryService
	noteService    *service.NoteService
	dataDir       string
)

func main() {
	// Determine data directory
	exe, err := os.Executable()
	if err != nil {
		dataDir = "./data"
	} else {
		dataDir = filepath.Join(filepath.Dir(exe), "data")
	}

	// Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("Warning: failed to load config: %v, using defaults", err)
		cfg = &config.Config{}
	}
	cfg.App.DataDir = dataDir

	// Initialize database
	if err := sqlite.InitDB(dataDir); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer sqlite.CloseDB()

	// Initialize services
	rssService = service.NewRSSService()
	filterService = service.NewFilterService()
	summaryService = service.NewSummaryService()

	notesDir := filepath.Join(dataDir, "notes")
	noteService = service.NewNoteService(notesDir)
	if err := noteService.Init(); err != nil {
		log.Printf("Warning: failed to initialize note service: %v", err)
	}

	// Initialize AI provider
	ai.InitProvider(cfg.AIProvider)

	// Setup routes
	mux := http.NewServeMux()

	// Feeds
	mux.HandleFunc("GET /api/feeds", handleGetFeeds)
	mux.HandleFunc("POST /api/feeds", handleAddFeed)
	mux.HandleFunc("PATCH /api/feeds/{id}", handleUpdateFeed)
	mux.HandleFunc("DELETE /api/feeds/{id}", handleDeleteFeed)
	mux.HandleFunc("GET /api/feeds/dead", handleGetDeadFeeds)
	mux.HandleFunc("DELETE /api/feeds/dead/{id}", handleDeleteDeadFeed)
	mux.HandleFunc("POST /api/feeds/{id}/refresh", handleRefreshFeed)
	mux.HandleFunc("POST /api/refresh", handleRefreshAllFeeds)

	// Articles
	mux.HandleFunc("GET /api/articles", handleGetArticles)
	mux.HandleFunc("GET /api/articles/search", handleSearchArticles)
	mux.HandleFunc("GET /api/articles/{id}", handleGetArticle)
	mux.HandleFunc("POST /api/articles/{id}/refresh", handleRefreshArticle)
	mux.HandleFunc("POST /api/articles/{id}/accept", handleAcceptArticle)
	mux.HandleFunc("POST /api/articles/{id}/reject", handleRejectArticle)
	mux.HandleFunc("POST /api/articles/{id}/snooze", handleSnoozeArticle)
	mux.HandleFunc("POST /api/articles/{id}/summary", handleGenerateSummary)
	mux.HandleFunc("POST /api/articles/{id}/note", handleCreateNote)
	mux.HandleFunc("POST /api/articles/{id}/filter", handleFilterArticle)

	// Filter rules
	mux.HandleFunc("GET /api/filter-rules", handleGetFilterRules)
	mux.HandleFunc("POST /api/filter-rules", handleAddFilterRule)
	mux.HandleFunc("DELETE /api/filter-rules/{id}", handleDeleteFilterRule)

	// Notes
	mux.HandleFunc("GET /api/notes", handleGetNotes)
	mux.HandleFunc("GET /api/notes/{id}", handleReadNote)
	mux.HandleFunc("DELETE /api/notes/{id}", handleDeleteNote)

	// AI Config
	mux.HandleFunc("GET /api/ai-config", handleGetAIConfig)
	mux.HandleFunc("PUT /api/ai-config", handleSaveAIConfig)

	// Health check
	mux.HandleFunc("GET /health", handleHealth)

	// OPML
	mux.HandleFunc("GET /opml", handleExportOPML)
	mux.HandleFunc("POST /opml", handleImportOPML)

	// SSE events stream
	mux.HandleFunc("GET /api/events", handleSSEvents)

	// CORS middleware
	handler := corsMiddleware(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
		log.Println("Server stopped")
	}()

	// Start background feed refresh if configured
	if cfg.Cron.Enabled && cfg.Cron.IntervalMins > 0 {
		interval := time.Duration(cfg.Cron.IntervalMins) * time.Minute
		go func() {
			// Wait a bit before first refresh so server is fully ready
			time.Sleep(30 * time.Second)
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for range ticker.C {
				log.Printf("[cron] Refreshing all feeds (interval: %v)", interval)
				if err := rssService.RefreshAllFeeds(); err != nil {
					log.Printf("[cron] RefreshAllFeeds error: %v", err)
					continue
				}
				newArticleIDs, err := filterService.FilterAllArticlesNew()
				if err != nil {
					log.Printf("[cron] FilterAllArticlesNew error: %v", err)
					continue
				}
				events.GlobalBroadcaster.Broadcast(events.EventNewArticles, map[string]int{"count": len(newArticleIDs)})
				go func() {
					summaryService.BatchGenerateSummaries(newArticleIDs, 5)
				}()
			}
		}()
		log.Printf("[cron] Feed refresh enabled (every %v)", interval)
	}

	log.Printf("Server starting on :%s", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

// ─── Middleware ────────────────────────────────────────────────────────────────

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func parseID(path string, r *http.Request) (int64, bool) {
	// Simple path param extraction: /api/feeds/123 → "123"
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, path), "/")
	if len(parts) < 2 {
		return 0, false
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

func parseArticleID(path string, r *http.Request) (int64, bool) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, path), "/")
	if len(parts) < 2 {
		return 0, false
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

func parseQueryInt(r *http.Request, key string, defaultVal int64) int64 {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultVal
	}
	i, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return defaultVal
	}
	return i
}

func readJSON(w http.ResponseWriter, r *http.Request, v interface{}) bool {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		return false
	}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(v); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return false
	}
	return true
}

// ─── Feed Handlers ────────────────────────────────────────────────────────────

func handleGetFeeds(w http.ResponseWriter, r *http.Request) {
	feeds, _ := rssService.GetFeeds()
	writeJSON(w, http.StatusOK, feeds)
}

func handleAddFeed(w http.ResponseWriter, r *http.Request) {
	var req struct{ URL string }
	if !readJSON(w, r, &req) {
		return
	}
	feed, err := rssService.AddFeed(req.URL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, feed)
}

func handleDeleteFeed(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID("/api/feeds", r)
	if !ok {
		http.Error(w, "invalid feed id", http.StatusBadRequest)
		return
	}
	if err := rssService.DeleteFeed(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleUpdateFeed(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID("/api/feeds", r)
	if !ok {
		http.Error(w, "invalid feed id", http.StatusBadRequest)
		return
	}
	var req struct {
		Group string `json:"group"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	feed, err := rssService.GetFeed(id)
	if err != nil {
		http.Error(w, "feed not found", http.StatusNotFound)
		return
	}
	feed.Group = req.Group
	if err := rssService.UpdateFeed(feed); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, feed)
}

func handleGetDeadFeeds(w http.ResponseWriter, r *http.Request) {
	feeds, _ := rssService.GetDeadFeeds()
	writeJSON(w, http.StatusOK, feeds)
}

func handleDeleteDeadFeed(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID("/api/feeds/dead", r)
	if !ok {
		http.Error(w, "invalid feed id", http.StatusBadRequest)
		return
	}
	if err := rssService.DeleteFeed(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleRefreshFeed(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID("/api/feeds", r)
	if !ok {
		http.Error(w, "invalid feed id", http.StatusBadRequest)
		return
	}
	if err := rssService.RefreshFeed(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleRefreshAllFeeds(w http.ResponseWriter, r *http.Request) {
	if err := rssService.RefreshAllFeeds(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	newArticleIDs, err := filterService.FilterAllArticlesNew()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Broadcast to SSE clients so they can refresh their article list
	events.GlobalBroadcaster.Broadcast(events.EventNewArticles, map[string]int{"count": len(newArticleIDs)})
	// Launch background goroutine to generate summaries
	go func() {
		summaryService.BatchGenerateSummaries(newArticleIDs, 5)
	}()
	w.WriteHeader(http.StatusNoContent)
}

// ─── Article Handlers ─────────────────────────────────────────────────────────

func handleGetArticles(w http.ResponseWriter, r *http.Request) {
	feedID := parseQueryInt(r, "feedId", 0)
	filterMode := r.URL.Query().Get("filterMode")
	if filterMode == "" {
		filterMode = "all"
	}
	limit := int(parseQueryInt(r, "limit", 100))
	offset := int(parseQueryInt(r, "offset", 0))
	articles, _ := rssService.GetArticles(feedID, filterMode, limit, offset)
	writeJSON(w, http.StatusOK, articles)
}

func handleSearchArticles(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeJSON(w, http.StatusOK, []models.Article{})
		return
	}
	articles, err := rssService.SearchArticles(q, 20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, articles)
}

func handleGetArticle(w http.ResponseWriter, r *http.Request) {
	id, ok := parseArticleID("/api/articles", r)
	if !ok {
		http.Error(w, "invalid article id", http.StatusBadRequest)
		return
	}
	article, err := rssService.GetArticle(id)
	if err != nil || article == nil {
		http.Error(w, "article not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, article)
}

func handleRefreshArticle(w http.ResponseWriter, r *http.Request) {
	id, ok := parseArticleID("/api/articles", r)
	if !ok {
		http.Error(w, "invalid article id", http.StatusBadRequest)
		return
	}
	article, err := rssService.RefreshArticle(id)
	if err != nil || article == nil {
		http.Error(w, "article not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, article)
}

func handleAcceptArticle(w http.ResponseWriter, r *http.Request) {
	id, ok := parseArticleID("/api/articles", r)
	if !ok {
		http.Error(w, "invalid article id", http.StatusBadRequest)
		return
	}
	if err := rssService.SetArticleStatus(id, "accepted"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleRejectArticle(w http.ResponseWriter, r *http.Request) {
	id, ok := parseArticleID("/api/articles", r)
	if !ok {
		http.Error(w, "invalid article id", http.StatusBadRequest)
		return
	}
	if err := rssService.SetArticleStatus(id, "rejected"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleSnoozeArticle(w http.ResponseWriter, r *http.Request) {
	id, ok := parseArticleID("/api/articles", r)
	if !ok {
		http.Error(w, "invalid article id", http.StatusBadRequest)
		return
	}
	if err := rssService.SetArticleStatus(id, "snoozed"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleGenerateSummary(w http.ResponseWriter, r *http.Request) {
	id, ok := parseArticleID("/api/articles", r)
	if !ok {
		http.Error(w, "invalid article id", http.StatusBadRequest)
		return
	}
	summary, err := summaryService.GenerateSummaryForArticle(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"summary": summary})
}

func handleCreateNote(w http.ResponseWriter, r *http.Request) {
	id, ok := parseArticleID("/api/articles", r)
	if !ok {
		http.Error(w, "invalid article id", http.StatusBadRequest)
		return
	}
	var req struct{ Summary string }
	if !readJSON(w, r, &req) {
		return
	}
	article, err := rssService.GetArticle(id)
	if err != nil {
		http.Error(w, "article not found", http.StatusNotFound)
		return
	}
	note, err := noteService.CreateNote(article, req.Summary)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, note)
}

func handleFilterArticle(w http.ResponseWriter, r *http.Request) {
	id, ok := parseArticleID("/api/articles", r)
	if !ok {
		http.Error(w, "invalid article id", http.StatusBadRequest)
		return
	}
	article, err := rssService.GetArticle(id)
	if err != nil {
		http.Error(w, "article not found", http.StatusNotFound)
		return
	}
	passed, err := filterService.FilterArticle(article)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"passed": passed})
}

// ─── Filter Rule Handlers ─────────────────────────────────────────────────────

func handleGetFilterRules(w http.ResponseWriter, r *http.Request) {
	rules, _ := filterService.GetRules()
	writeJSON(w, http.StatusOK, rules)
}

func handleAddFilterRule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type   string `json:"type"`
		Value  string `json:"value"`
		Action string `json:"action"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if err := filterService.AddRule(req.Type, req.Value, req.Action); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func handleDeleteFilterRule(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID("/api/filter-rules", r)
	if !ok {
		http.Error(w, "invalid rule id", http.StatusBadRequest)
		return
	}
	if err := filterService.DeleteRule(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── Note Handlers ────────────────────────────────────────────────────────────

func handleGetNotes(w http.ResponseWriter, r *http.Request) {
	notes, _ := noteService.GetNotes()
	writeJSON(w, http.StatusOK, notes)
}

func handleReadNote(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID("/api/notes", r)
	if !ok {
		http.Error(w, "invalid note id", http.StatusBadRequest)
		return
	}
	note, err := noteService.GetNoteByArticleID(id)
	if err != nil {
		http.Error(w, "note not found", http.StatusNotFound)
		return
	}
	content, err := noteService.ReadNote(note)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"content": content})
}

func handleDeleteNote(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID("/api/notes", r)
	if !ok {
		http.Error(w, "invalid note id", http.StatusBadRequest)
		return
	}
	if err := noteService.DeleteNote(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── AI Config Handlers ───────────────────────────────────────────────────────

func handleGetAIConfig(w http.ResponseWriter, r *http.Request) {
	cfg := config.AppConfig_
	if cfg == nil {
		cfg, _ = config.LoadConfig()
	}
	aiConfig := models.AIProviderConfig{
		Provider:  cfg.AIProvider.Provider,
		APIKey:    cfg.AIProvider.APIKey,
		BaseURL:   cfg.AIProvider.BaseURL,
		Model:     cfg.AIProvider.Model,
		MaxTokens: cfg.AIProvider.MaxTokens,
	}
	writeJSON(w, http.StatusOK, aiConfig)
}

func handleSaveAIConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider  string `json:"provider"`
		APIKey   string `json:"api_key"`
		BaseURL  string `json:"base_url"`
		Model    string `json:"model"`
		MaxTokens int    `json:"max_tokens"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	cfg := config.AppConfig_
	if cfg == nil {
		cfg, _ = config.LoadConfig()
	}
	cfg.AIProvider = config.AIProviderConfig{
		Provider:  req.Provider,
		APIKey:   req.APIKey,
		BaseURL:  req.BaseURL,
		Model:    req.Model,
		MaxTokens: req.MaxTokens,
	}
	if err := config.SaveConfig(cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ai.InitProvider(cfg.AIProvider)
	w.WriteHeader(http.StatusNoContent)
}

// ─── Health Check ─────────────────────────────────────────────────────────────

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	// Check DB connectivity
	if sqlite.DB == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "down", "db": "no connection"})
		return
	}
	if err := sqlite.DB.Ping(); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "down", "db": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "db": "connected"})
}

// ─── OPML ───────────────────────────────────────────────────────────────────

func handleExportOPML(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	feeds, err := rssService.GetFeeds()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data, err := opml.Export(feeds)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="feeds.opml"`)
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func handleImportOPML(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.Header.Get("Content-Type") != "application/xml" && r.Header.Get("Content-Type") != "text/xml" {
		http.Error(w, "Content-Type must be application/xml", http.StatusBadRequest)
		return
	}
	urls, err := opml.Import(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(urls) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"imported": 0, "message": "no feeds found in OPML"})
		return
	}
	added := 0
	for _, url := range urls {
		_, err := rssService.AddFeed(url)
		if err == nil {
			added++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"imported": added, "total": len(urls)})
}

// ─── SSE Events ───────────────────────────────────────────────────────────────

func handleSSEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	ch := events.GlobalBroadcaster.Add()
	defer events.GlobalBroadcaster.Remove(ch)

	// Send initial ping
	fmt.Fprintf(w, "event: ping\ndata: {}\n\n")
	flusher.Flush()

	// Keep connection alive, drain messages
	for {
		select {
		case data, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
