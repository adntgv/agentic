package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aid/agentic/internal/config"
	"github.com/aid/agentic/internal/handler"
	"github.com/aid/agentic/internal/middleware"
	"github.com/aid/agentic/internal/repository"
	"github.com/aid/agentic/internal/s3"
	"github.com/aid/agentic/internal/service"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// Connect to database
	dbPool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer dbPool.Close()

	// Initialize S3 client
	s3Client, err := s3.NewClient(cfg.S3Endpoint, cfg.S3Bucket, cfg.S3AccessKey, cfg.S3SecretKey, "us-east-1")
	_ = s3Client // TODO: wire into artifact service
	if err != nil {
		log.Fatalf("init s3 client: %v", err)
	}

	// Initialize repositories
	taskRepo := repository.NewTaskRepo(dbPool)
	bidRepo := repository.NewBidRepo(dbPool)
	escrowRepo := repository.NewEscrowRepo(dbPool)
	outboxRepo := repository.NewOutboxRepo(dbPool)
	disputeRepo := repository.NewDisputeRepo(dbPool)
	bondRepo := repository.NewDisputeBondRepo(dbPool)
	workerRepo := repository.NewWorkerRepo(dbPool)
	userRepo := repository.NewUserRepo(dbPool)
	agentRepo := repository.NewAgentRepo(dbPool)
	artifactRepo := repository.NewArtifactRepo(dbPool)
	repRepo := repository.NewReputationRepo(dbPool)

	// Initialize services
	escrowSvc := service.NewEscrowService(escrowRepo, nil, outboxRepo) // txRepo to be added
	schedulerSvc := service.NewSchedulerService(nil, nil) // will be wired after taskSvc and disputeSvc
	
	taskSvc := service.NewTaskService(taskRepo, bidRepo, escrowRepo, outboxRepo, escrowSvc, schedulerSvc)
	bidSvc := service.NewBidService(bidRepo, taskRepo, outboxRepo)
	disputeSvc := service.NewDisputeService(disputeRepo, bondRepo, taskRepo, escrowRepo, outboxRepo)
	workerSvc := service.NewWorkerService(workerRepo, userRepo, agentRepo)
	artifactSvc := service.NewArtifactService(artifactRepo)
	repSvc := service.NewReputationService(repRepo)
	_ = repSvc // TODO: wire into handler

	// Wire scheduler
	schedulerSvc = service.NewSchedulerService(taskSvc, disputeSvc)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(workerSvc)
	taskHandler := handler.NewTaskHandler(taskSvc)
	bidHandler := handler.NewBidHandler(bidSvc)
	workHandler := handler.NewWorkHandler(taskSvc)
	artifactHandler := handler.NewArtifactHandler(artifactSvc)
	disputeHandler := handler.NewDisputeHandler(disputeSvc)
	escrowHandler := handler.NewEscrowHandler(escrowSvc)
	workerHandler := handler.NewWorkerHandler(workerSvc)
	webhookHandler := handler.NewWebhookHandler()
	adminHandler := handler.NewAdminHandler(taskSvc)

	// Setup router
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.CORS)

	// Auth endpoints (no auth required)
	r.Post("/auth/wallet", authHandler.WalletAuth)
	r.Post("/auth/apikey", authHandler.APIKeyAuth)

	// Protected routes (require auth)
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth)
		r.Use(middleware.Idempotency)

		// Tasks
		r.Post("/tasks", taskHandler.Create)
		r.Get("/tasks", taskHandler.List)
		r.Get("/tasks/{id}", taskHandler.Get)
		r.Patch("/tasks/{id}", taskHandler.Update)
		r.Delete("/tasks/{id}", taskHandler.Delete)
		r.Post("/tasks/{id}/cancel", taskHandler.Cancel)

		// Bids
		r.Post("/tasks/{id}/bids", bidHandler.Create)
		r.Get("/tasks/{id}/bids", bidHandler.List)
		r.Post("/tasks/{id}/bids/{bidId}/accept", bidHandler.Accept)

		// Work
		r.Post("/tasks/{id}/ack", workHandler.Ack)
		r.Post("/tasks/{id}/submit", workHandler.Submit)
		r.Post("/tasks/{id}/approve", workHandler.Approve)
		r.Post("/tasks/{id}/revision", workHandler.Revision)

		// Artifacts
		r.Post("/tasks/{id}/artifacts/upload-url", artifactHandler.UploadURL)
		r.Get("/tasks/{id}/artifacts", artifactHandler.List)

		// Escrow
		r.Get("/tasks/{id}/escrow", escrowHandler.Get)

		// Disputes
		r.Post("/tasks/{id}/disputes", disputeHandler.Raise)
		r.Post("/disputes/{id}/respond", disputeHandler.Respond)
		r.Post("/disputes/{id}/evidence", disputeHandler.Evidence)
		r.Post("/disputes/{id}/ruling", disputeHandler.Ruling)

		// Workers
		r.Get("/workers/{id}", workerHandler.Get)
		r.Get("/workers/{id}/history", workerHandler.History)
		r.Get("/operators/{id}", workerHandler.GetOperator)

		// Webhooks
		r.Post("/webhooks", webhookHandler.Create)
		r.Get("/webhooks", webhookHandler.List)

		// Admin
		r.Post("/tasks/{id}/unassign", adminHandler.Unassign)
	})

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	// Start scheduler in background
	go func() {
		if err := schedulerSvc.Run(context.Background()); err != nil {
			log.Printf("scheduler error: %v", err)
		}
	}()

	// Start HTTP server
	port := cfg.Port
	if port == "" {
		port = ":8080"
	}

	server := &http.Server{
		Addr:         port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		log.Println("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server forced to shutdown: %v", err)
		}

		dbPool.Close()
	}()

	log.Printf("Starting server on %s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}

	log.Println("Server stopped")
}
