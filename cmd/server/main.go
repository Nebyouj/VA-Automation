package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/nebyoudamtew/go-job-queue/interval/database"
	"github.com/nebyoudamtew/go-job-queue/interval/executor"
	filesIntegration "github.com/nebyoudamtew/go-job-queue/interval/infrastructure/files"
	researchIntegration "github.com/nebyoudamtew/go-job-queue/interval/infrastructure/research"
	calendarIntegration "github.com/nebyoudamtew/go-job-queue/interval/integrations/calendar"
	"github.com/nebyoudamtew/go-job-queue/interval/integrations/email"
	sheetsIntegration "github.com/nebyoudamtew/go-job-queue/interval/integrations/sheets"
	"github.com/nebyoudamtew/go-job-queue/interval/interfaces/http/handlers"

	"github.com/nebyoudamtew/go-job-queue/interval/application/jobs"
	"github.com/nebyoudamtew/go-job-queue/interval/infrastructure/postgres"
	"github.com/nebyoudamtew/go-job-queue/interval/infrastructure/queue"
)

type researchAdapter struct {
	crawler *researchIntegration.Crawler
}

func (a researchAdapter) Fetch(
	ctx context.Context,
	rawURL string,
) (executor.ResearchPage, error) {
	page, err := a.crawler.Fetch(ctx, rawURL)
	if err != nil {
		return executor.ResearchPage{}, err
	}

	return executor.ResearchPage{
		URL:   page.URL,
		Title: page.Title,
		Text:  page.Text,
	}, nil
}

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("warning: .env file not found")
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	databaseURL := os.Getenv("DATABASE_URL")

	if databaseURL == "" {
		fmt.Println("DATABASE_URL is not set")
		return
	}

	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUsername := os.Getenv("SMTP_USERNAME")
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	smtpFrom := os.Getenv("SMTP_FROM")

	if smtpHost == "" ||
		smtpPort == "" ||
		smtpUsername == "" ||
		smtpPassword == "" ||
		smtpFrom == "" {
		fmt.Println("SMTP configuration is incomplete")
		return
	}

	emailService := email.NewSMTPEmailService(
		smtpHost,
		smtpPort,
		smtpUsername,
		smtpPassword,
		smtpFrom,
	)

	emailRenderer := email.NewTemplateRenderer(
		"interval/templates/email",
	)

	db, err := database.NewPool(ctx, databaseURL)
	if err != nil {
		fmt.Println("database connection failed:", err)
		return
	}
	defer db.Close()

	fmt.Println("Database connected")

	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	googleRedirectURL := os.Getenv("GOOGLE_REDIRECT_URL")

	if googleClientID == "" || googleClientSecret == "" || googleRedirectURL == "" {
		log.Fatal("GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, and GOOGLE_REDIRECT_URL must be configured")
	}

	tokenPath := filepath.Join(".", "calendar_token.json")

	oauthManager := calendarIntegration.NewOAuthManager(
		googleClientID,
		googleClientSecret,
		googleRedirectURL,
		tokenPath,
	)

	calendarService, err := calendarIntegration.NewGoogleCalendarService(
		context.Background(),
		oauthManager,
	)
	if err != nil {
		log.Fatalf("Failed to initialize Google Calendar service: %v", err)
	}

	googleSheetsService, err := sheetsIntegration.NewGoogleSheetsService(
		context.Background(),
		oauthManager,
	)
	if err != nil {
		log.Fatalf("Failed to initialize Google Sheets service: %v", err)
	}

	fileProcessor, err := filesIntegration.NewLocalFileProcessor(
		"data/input",
		"data/output",
	)
	if err != nil {
		log.Fatalf("Failed to initialize file processor: %v", err)
	}

	q := queue.NewQueue(10)

	executor := executor.NewExecutor(
		emailService,
		&executor.StubReportGenerator{},
		fileProcessor,
		calendarService,
		googleSheetsService,
		emailRenderer,
	)

	crawler := researchIntegration.NewCrawler()
	executor.SetResearchService(researchAdapter{
		crawler: crawler,
	})

	jobRepository := postgres.NewJobRepository(db)

	jobService := jobs.NewService(q, jobRepository)

	q.StartWorker(ctx, 1, executor, jobService)
	q.StartWorker(ctx, 2, executor, jobService)
	q.StartWorker(ctx, 3, executor, jobService)

	if err := jobService.RecoverJobs(ctx); err != nil {
		fmt.Println("job recovery failed:", err)
		return
	}

	jobHandler := handlers.NewJobHandler(jobService)

	cors := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next(w, r)
		}
	}

	http.HandleFunc("/jobs", cors(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			jobHandler.CreateJob(w, r)

		case http.MethodGet:
			jobHandler.GetJobs(w, r)

		default:
			http.Error(
				w,
				"method not allowed",
				http.StatusMethodNotAllowed,
			)
		}
	}))

	http.HandleFunc("/jobs/{id}", cors(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			jobHandler.GetJob(w, r)

		case http.MethodPost:
			jobHandler.CancelJob(w, r)

		default:
			http.Error(
				w,
				"method not allowed",
				http.StatusMethodNotAllowed,
			)
		}
	}))

	http.HandleFunc("GET /oauth/google", oauthManager.StartHandler)
	http.HandleFunc("GET /oauth/google/callback", oauthManager.CallbackHandler)

	http.HandleFunc("/stats", cors(jobHandler.GetStats))

	server := &http.Server{
		Addr: ":8080",
	}

	go func() {
		fmt.Println("Server running on http://localhost:8080")

		if err := server.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {
			fmt.Println("server error:", err)
			stop()
		}
	}()
	<-ctx.Done()

	fmt.Println("Shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		fmt.Println("server shutdown error:", err)
	}

	q.Close()

	if q.Wait(shutdownCtx) {
		fmt.Println("All workers stopped")
	} else {
		fmt.Println("Shutdown timeout reached; workers are still running")
	}

	fmt.Println("Server stopped")
}
