package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Rizwank123/notification_system/internal/config"
	"github.com/Rizwank123/notification_system/internal/database"
	"github.com/Rizwank123/notification_system/internal/delivery"
	"github.com/Rizwank123/notification_system/internal/repository"
	"github.com/Rizwank123/notification_system/internal/service"
	"github.com/Rizwank123/notification_system/pkg/logger"
)

func main() {
	// Initialize logger
	appLogger := logger.New()
	appLogger.Info("Starting Notification System...")

	// Load configuration using Viper with mapstructure
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	appLogger.Info("Configuration loaded",
		"environment", cfg.AppEnvironment,
		"app_name", cfg.AppName,
		"db_host", cfg.DatabaseHost,
		"db_name", cfg.DatabaseName,
		"workers", cfg.QueueWorkers,
	)

	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database connection
	db, err := database.NewPostgresDB(ctx, database.Config{
		Host:     cfg.DatabaseHost,
		Port:     cfg.DatabasePort,
		User:     cfg.DatabaseUsername,
		Password: cfg.DatabasePassword,
		DBName:   cfg.DatabaseName,
		SSLMode:  cfg.DatabaseSSLMode,
		MaxConns: cfg.DatabaseMaxConns,
		MinConns: cfg.DatabaseMinConns,
	}, appLogger)
	if err != nil {
		appLogger.Fatal("Failed to connect to database", "error", err)
	}
	defer db.Close()

	// Verify database connection
	if err := db.Ping(ctx); err != nil {
		appLogger.Fatal("Database ping failed", "error", err)
	}

	stats := db.Stats()
	appLogger.Info("Database connection verified",
		"total_conns", stats["total_conns"],
		"idle_conns", stats["idle_conns"],
	)

	// ✅ FIXED: Initialize Firebase service PROPERLY
	var firebaseService *delivery.FirebaseService
	if cfg.FirebaseProjectID != "" && cfg.FirebaseCredentialPath != "" {
		appLogger.Info("🔥 Initializing Firebase service...",
			"project_id", cfg.FirebaseProjectID,
			"cred_path", cfg.FirebaseCredentialPath,
		)

		firebaseService, err = delivery.NewFirebaseService(ctx, cfg, appLogger)
		if err != nil {
			appLogger.Error("❌ Failed to initialize Firebase, using simulated delivery", "error", err)
			firebaseService = nil
		} else {
			appLogger.Info("✅ Firebase service initialized successfully!")
		}
	} else {
		appLogger.Error("⚠️ Firebase not configured (missing PROJECT_ID or CRED_PATH), using simulated delivery")
	}
	// Initialize repositories
	notificationRepo := repository.NewNotificationRepository(db.Pool(), appLogger)

	// Initialize notification service with updated config structure
	notificationService := service.NewNotificationService(cfg, appLogger, notificationRepo, firebaseService)

	// Start the service
	if err := notificationService.Start(ctx); err != nil {
		appLogger.Fatal("Failed to start notification service", "error", err)
	}

	appLogger.Info(fmt.Sprintf("Server started on port %d", cfg.AppPort))

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	appLogger.Info("Shutting down gracefully...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := notificationService.Shutdown(shutdownCtx); err != nil {
		appLogger.Error("Error during shutdown", "error", err)
	}

	appLogger.Info("Notification system stopped")
}
