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

	"github.com/robfig/cron/v3"
)

// Global services (same as main.go)
var (
	rssService      *service.RSSService
	filterService   *service.FilterService
	summaryService  *service.SummaryService
	noteService     *service.NoteService
	briefingService *service.BriefingService
	dataDir         string
)

func main() {
	// Determine data directory - use fixed path for development
	dataDir = "./data"

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

	// Initialize services
	rssService = service.NewRSSService()
	filterService = service.NewFilterService()
	summaryService = service.NewSummaryService()

	notesDir := filepath.Join(dataDir, "notes")
	noteService = service.NewNoteService(notesDir)
	if err := noteService.Init(); err != nil {
		log.Printf("Warning: failed to initialize note service: %v", err)
	}

	// Initialize briefing service
	briefingService = service.NewBriefingService()

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
	mux.HandleFunc("GET /api/refresh/status", handleGetRefreshStatus)
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

	// Briefings
	mux.HandleFunc("GET /api/briefings", handleGetBriefings)
	mux.HandleFunc("GET /api/briefings/{id}", handleGetBriefing)
	mux.HandleFunc("POST /api/briefings/generate", handleGenerateBriefing)
	mux.HandleFunc("DELETE /api/briefings/{id}", handleDeleteBriefing)
	mux.HandleFunc("GET /api/briefings/{id}/status", handleGetBriefingStatus)

	// AI Config
	mux.HandleFunc("GET /api/ai-config", handleGetAIConfig)
	mux.HandleFunc("PUT /api/ai-config", handleSaveAIConfig)
	mux.HandleFunc("POST /api/ai-config/test", handleTestAIConfig)

	// Health check
	mux.HandleFunc("GET /health", handleHealth)

	// OPML
	mux.HandleFunc("GET /opml", handleExportOPML)
	mux.HandleFunc("POST /opml", handleImportOPML)

	// Stats
	mux.HandleFunc("GET /api/stats", handleStats)

	// Export
	mux.HandleFunc("GET /api/export", handleExport)

	// SSE events stream
	mux.HandleFunc("GET /api/events", handleSSEvents)

	// CORS middleware
	handler := corsMiddleware(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5562"
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

	// Start background briefing cron if configured
	if cfg.Cron.Enabled {
		c := cron.New()
		// Run at specific time each day: "Minute Hour * * *"
		schedule := fmt.Sprintf("%d %d * * *", cfg.Cron.Minute, cfg.Cron.Hour)
		c.AddFunc(schedule, func() {
			log.Printf("[cron] Daily briefing at %02d:%02d - refreshing feeds first", cfg.Cron.Hour, cfg.Cron.Minute)
			if err := rssService.RefreshAllFeeds(); err != nil {
				log.Printf("[cron] RefreshAllFeeds error: %v", err)
			}
			log.Printf("[cron] Generating daily briefing")
			briefingService.GenerateBriefing()
		})
		c.Start()
		log.Printf("[cron] Briefing scheduled at %02d:%02d daily", cfg.Cron.Hour, cfg.Cron.Minute)
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

func handleGetRefreshStatus(w http.ResponseWriter, r *http.Request) {
	events.GlobalRefreshStatus.Mutex.Lock()
	defer events.GlobalRefreshStatus.Mutex.Unlock()

	if !events.GlobalRefreshStatus.InProgress {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"inProgress": false,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"inProgress": true,
		"current":    events.GlobalRefreshStatus.Current,
		"total":      events.GlobalRefreshStatus.Total,
		"feedTitle":  events.GlobalRefreshStatus.FeedTitle,
		"success":    events.GlobalRefreshStatus.Success,
		"failed":     events.GlobalRefreshStatus.Failed,
		"error":      events.GlobalRefreshStatus.Error,
	})
}

func handleRefreshAllFeeds(w http.ResponseWriter, r *http.Request) {
	// Check operation mutex - return 409 if another operation is in progress
	if !events.GlobalOperationState.TryLock("refreshing") {
		writeJSON(w, http.StatusConflict, map[string]interface{}{
			"success": false,
			"error":   "正在刷新订阅源，请稍候",
			"code":    "OPERATION_IN_PROGRESS",
		})
		return
	}

	// Return 202 Accepted immediately
	taskID := fmt.Sprintf("refresh-%d", time.Now().UnixMilli())
	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"success": true,
		"taskId":  taskID,
	})

	// Spawn goroutine for actual work
	go func() {
		defer events.GlobalOperationState.Unlock()

		feeds, _ := rssService.GetFeeds()
		total := len(feeds)

		// Update status: start
		events.GlobalRefreshStatus.Mutex.Lock()
		events.GlobalRefreshStatus.InProgress = true
		events.GlobalRefreshStatus.Current = 0
		events.GlobalRefreshStatus.Total = total
		events.GlobalRefreshStatus.FeedTitle = ""
		events.GlobalRefreshStatus.Success = 0
		events.GlobalRefreshStatus.Failed = 0
		events.GlobalRefreshStatus.Error = ""
		events.GlobalRefreshStatus.Mutex.Unlock()

		// Refresh with progress callback
		err := rssService.RefreshAllFeedsWithProgress(func(idx, total int, feedTitle string, feedId int64, newCount int, errMsg string) {
			events.GlobalRefreshStatus.Mutex.Lock()
			events.GlobalRefreshStatus.Current = idx // 已完成数量
			events.GlobalRefreshStatus.Total = total
			events.GlobalRefreshStatus.FeedTitle = feedTitle
			if errMsg != "" {
				events.GlobalRefreshStatus.Failed++
			} else {
				events.GlobalRefreshStatus.Success++
			}
			events.GlobalRefreshStatus.Error = errMsg
			events.GlobalRefreshStatus.Mutex.Unlock()
		})

		if err != nil {
			events.GlobalRefreshStatus.Mutex.Lock()
			events.GlobalRefreshStatus.InProgress = false
			events.GlobalRefreshStatus.Error = err.Error()
			events.GlobalRefreshStatus.Mutex.Unlock()
			return
		}

		// Filter new articles
		newArticleIDs, filterErr := filterService.FilterAllArticlesNew()
		if filterErr != nil {
			events.GlobalRefreshStatus.Mutex.Lock()
			events.GlobalRefreshStatus.InProgress = false
			events.GlobalRefreshStatus.Error = filterErr.Error()
			events.GlobalRefreshStatus.Mutex.Unlock()
			return
		}

		// Update status: complete
		events.GlobalRefreshStatus.Mutex.Lock()
		events.GlobalRefreshStatus.InProgress = false
		events.GlobalRefreshStatus.Success = len(newArticleIDs)
		events.GlobalRefreshStatus.Failed = total - len(newArticleIDs)
		events.GlobalRefreshStatus.Mutex.Unlock()
		events.GlobalBroadcaster.Broadcast(events.EventRefreshComplete, events.RefreshComplete{
			Success: len(newArticleIDs),
			Failed:  total - len(newArticleIDs),
		})

		// Broadcast new articles event so frontend can refresh list
		events.GlobalBroadcaster.Broadcast(events.EventNewArticles, map[string]int{"count": len(newArticleIDs)})

		// Launch background goroutine to generate summaries
		go func() {
			summaryService.BatchGenerateSummaries(newArticleIDs, 5)
		}()
	}()
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
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if article == nil {
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

// ─── Briefing Handlers ─────────────────────────────────────────────────────────

func handleGetBriefings(w http.ResponseWriter, r *http.Request) {
	limit := int(parseQueryInt(r, "limit", 20))
	offset := int(parseQueryInt(r, "offset", 0))
	briefings, _ := briefingService.GetAllBriefings(limit, offset)
	writeJSON(w, http.StatusOK, briefings)
}

func handleGetBriefing(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID("/api/briefings", r)
	if !ok {
		http.Error(w, "invalid briefing id", http.StatusBadRequest)
		return
	}
	briefing, err := briefingService.GetBriefingWithItems(id)
	if err != nil {
		http.Error(w, "briefing not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, briefing)
}

func handleGenerateBriefing(w http.ResponseWriter, r *http.Request) {
	// Check round-block logic
	if !briefingService.LastBriefingAt.Before(briefingService.LastRefreshAt) && !briefingService.LastRefreshAt.IsZero() {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   "本轮已生成简报，请稍后再试",
		})
		return
	}

	// Check operation mutex - return 409 if another operation is in progress
	if !events.GlobalOperationState.TryLock("generating") {
		writeJSON(w, http.StatusConflict, map[string]interface{}{
			"success": false,
			"error":   "正在生成简报，请稍候",
			"code":    "OPERATION_IN_PROGRESS",
		})
		return
	}

	// Return 202 Accepted immediately
	taskID := fmt.Sprintf("briefing-%d", time.Now().UnixMilli())
	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"success": true,
		"taskId":  taskID,
	})

	// Spawn goroutine for actual work
	go func() {
		defer events.GlobalOperationState.Unlock()

		// Broadcast start
		events.GlobalBroadcaster.Broadcast(events.EventBriefingStart, struct{}{})

		// 1. Refresh all feeds with progress
		feeds, _ := rssService.GetFeeds()
		total := len(feeds)
		events.GlobalBroadcaster.Broadcast(events.EventRefreshStart, map[string]int{"total": total})

		refreshErr := rssService.RefreshAllFeedsWithProgress(func(idx, total int, feedTitle string, feedId int64, newCount int, errMsg string) {
			events.GlobalBroadcaster.Broadcast(events.EventRefreshProgress, events.RefreshProgress{
				Current:   idx,
				Total:     total,
				FeedTitle: feedTitle,
				FeedId:    feedId,
				NewCount:  newCount,
				Error:     errMsg,
			})
		})

		if refreshErr != nil {
			events.GlobalBroadcaster.Broadcast(events.EventRefreshError, map[string]string{"message": refreshErr.Error()})
			events.GlobalBroadcaster.Broadcast(events.EventBriefingError, map[string]string{"message": refreshErr.Error()})
			return
		}

		events.GlobalBroadcaster.Broadcast(events.EventRefreshComplete, events.RefreshComplete{Success: total, Failed: 0})

		// 2. Generate briefing with progress
		briefing, err := briefingService.GenerateBriefingWithProgress(func(stage, detail string) {
			events.GlobalBroadcaster.Broadcast(events.EventBriefingProgress, events.BriefingProgress{
				Stage:  stage,
				Detail: detail,
			})
		})

		if err != nil {
			events.GlobalBroadcaster.Broadcast(events.EventBriefingError, map[string]string{"message": err.Error()})
			return
		}

		// 3. Only update LastRefreshAt after successful briefing
		briefingService.LastRefreshAt = time.Now()

		// Broadcast complete
		events.GlobalBroadcaster.Broadcast(events.EventBriefingComplete, events.BriefingComplete{
			BriefingID: briefing.ID,
		})
	}()
}

func handleDeleteBriefing(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID("/api/briefings", r)
	if !ok {
		http.Error(w, "invalid briefing id", http.StatusBadRequest)
		return
	}
	briefingService.DeleteBriefing(id)
	w.WriteHeader(http.StatusNoContent)
}

func handleGetBriefingStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID("/api/briefings", r)
	if !ok {
		http.Error(w, "invalid briefing id", http.StatusBadRequest)
		return
	}
	briefing, err := briefingService.GetBriefingWithItems(id)
	if err != nil {
		http.Error(w, "briefing not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":     briefing.Status,
		"error":      briefing.Error,
		"created_at": briefing.CreatedAt,
	})
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

func handleTestAIConfig(w http.ResponseWriter, r *http.Request) {
	// Just save the config without actually testing the connection
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Configuration saved!",
	})
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

// ─── Stats ───────────────────────────────────────────────────────────────────

func handleStats(w http.ResponseWriter, r *http.Request) {
	type feedStat struct {
		FeedID    int64  `json:"feed_id"`
		Title     string `json:"title"`
		Total     int    `json:"total"`
		Unread    int    `json:"unread"`
		Accepted  int    `json:"accepted"`
		Rejected  int    `json:"rejected"`
		Snoozed   int    `json:"snoozed"`
		Filtered  int    `json:"filtered"`
		Saved     int    `json:"saved"`
	}

	// Global counts: one query with CASE WHEN instead of 7 separate COUNT(*)
	var total, unread, accepted, rejected, snoozed, filtered, saved int
	err := sqlite.DB.QueryRow(`
		SELECT
			COUNT(*),
			COALESCE(SUM(CASE WHEN status = 'unread' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'accepted' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'rejected' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'snoozed' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN is_filtered = 1 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN is_saved = 1 THEN 1 ELSE 0 END), 0)
		FROM articles
	`).Scan(&total, &unread, &accepted, &rejected, &snoozed, &filtered, &saved)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	feeds, _ := rssService.GetFeeds()
	feedStats := make([]feedStat, 0, len(feeds))
	for _, f := range feeds {
		var fTotal, fUnread, fAccepted, fRejected, fSnoozed, fFiltered, fSaved int
		sqlite.DB.QueryRow(`
			SELECT
				COUNT(*),
				COALESCE(SUM(CASE WHEN status = 'unread' THEN 1 ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN status = 'accepted' THEN 1 ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN status = 'rejected' THEN 1 ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN status = 'snoozed' THEN 1 ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN is_filtered = 1 THEN 1 ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN is_saved = 1 THEN 1 ELSE 0 END), 0)
			FROM articles WHERE feed_id = ?
		`, f.ID).Scan(&fTotal, &fUnread, &fAccepted, &fRejected, &fSnoozed, &fFiltered, &fSaved)
		feedStats = append(feedStats, feedStat{
			FeedID: f.ID, Title: f.Title, Total: fTotal,
			Unread: fUnread, Accepted: fAccepted, Rejected: fRejected,
			Snoozed: fSnoozed, Filtered: fFiltered, Saved: fSaved,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"total_articles": total,
		"unread":        unread,
		"accepted":      accepted,
		"rejected":      rejected,
		"snoozed":       snoozed,
		"filtered":      filtered,
		"saved":         saved,
		"feeds":         feedStats,
	})
}

// ─── Export ───────────────────────────────────────────────────────────────────

func handleExport(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	articles, err := rssService.GetArticles(0, "saved", 0, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(articles) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"articles":[]}`))
		return
	}

	if format == "markdown" {
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="saved-articles.md"`)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("# Saved Articles\n\n"))
		for _, a := range articles {
			w.Write([]byte(fmt.Sprintf("## [%s](%s)\n\n", a.Title, a.Link)))
			w.Write([]byte(fmt.Sprintf("**Published:** %s\n\n", a.Published.Format("2006-01-02 15:04"))))
			if a.Summary != "" {
				w.Write([]byte(fmt.Sprintf("%s\n\n", a.Summary)))
			}
			if a.Content != "" {
				// Plain text: strip HTML tags roughly
				content := stripHTML(a.Content)
				w.Write([]byte(content + "\n\n"))
			}
			w.Write([]byte("---\n\n"))
		}
	} else {
		// JSON
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="saved-articles.json"`)
		writeJSON(w, http.StatusOK, map[string]interface{}{"articles": articles})
	}
}

// stripHTML removes basic HTML tags from content
func stripHTML(html string) string {
	result := ""
	depth := 0
	for _, r := range html {
		if r == '<' {
			depth++
		} else if r == '>' {
			depth--
		} else if depth == 0 {
			result += string(r)
		}
	}
	// Collapse whitespace
	lines := strings.Split(strings.TrimSpace(result), "\n")
	var clean []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			clean = append(clean, l)
		}
	}
	return strings.Join(clean, "\n")
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
	fmt.Fprintf(w, "event: ping\r\ndata: {}\r\n\r\n")
	flusher.Flush()

	// Keep connection alive, drain messages
	for {
		select {
		case data, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprint(w, data)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
