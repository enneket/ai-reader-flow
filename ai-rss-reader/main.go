package main

import (
	"embed"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"ai-rss-reader/internal/ai"
	"ai-rss-reader/internal/config"
	"ai-rss-reader/internal/repository/sqlite"
	"ai-rss-reader/internal/service"

	"github.com/robfig/cron/v3"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

var (
	rssService     *service.RSSService
	filterService  *service.FilterService
	summaryService *service.SummaryService
	noteService    *service.NoteService
	cronScheduler  *cron.Cron
	dataDir        string
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

	// Setup cron job for RSS refresh
	if cfg.Cron.Enabled {
		cronScheduler = cron.New()
		schedule := fmt.Sprintf("*/%d * * * *", cfg.Cron.IntervalMins)
		_, err := cronScheduler.AddFunc(schedule, func() {
			log.Println("Refreshing RSS feeds...")
			if err := rssService.RefreshAllFeeds(); err != nil {
				log.Printf("Error refreshing feeds: %v", err)
			}
			// Apply filters
			if err := filterService.FilterAllArticles(); err != nil {
				log.Printf("Error applying filters: %v", err)
			}
		})
		if err != nil {
			log.Printf("Warning: failed to setup cron job: %v", err)
		} else {
			cronScheduler.Start()
			log.Printf("RSS refresh scheduler started (every %d minutes)", cfg.Cron.IntervalMins)
		}
	}

	// Create the app
	app := NewApp()

	// Handle shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		if cronScheduler != nil {
			cronScheduler.Stop()
		}
		os.Exit(0)
	}()

	// Run the application
	err = wails.Run(&options.App{
		Title:  "AI RSS Reader",
		Width:  1280,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		log.Fatalf("Error running application: %v", err)
	}
}
