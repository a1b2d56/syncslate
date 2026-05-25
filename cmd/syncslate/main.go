package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"syncslate/internal/auth"
	"syncslate/internal/chat"
	"syncslate/internal/config"
	"syncslate/internal/contact"
	"syncslate/internal/database"
	"syncslate/internal/group"
	"syncslate/internal/health"
	"syncslate/internal/media"
	"syncslate/internal/message"
	"syncslate/internal/middleware"
	"syncslate/internal/request"
	"syncslate/internal/user"
	"syncslate/internal/ws"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	// 1. Config
	cfg := config.Load()

	// Logger setup
	opts := &slog.HandlerOptions{}
	if cfg.LogLevel == "debug" {
		opts.Level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, opts)))

	slog.Info("starting syncslate backend server", "port", cfg.Port)

	// 2. Database & Migrations
	db, err := database.New(cfg.DBPath, cfg.DBMaxReaders)
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}

	if err := db.RunMigrations("./migrations"); err != nil {
		slog.Error("failed to run database migrations", "error", err)
		os.Exit(1)
	}

	// 3. Services
	authService := auth.NewService(db.DB, cfg)
	userService := user.NewService(db.DB)
	contactService := contact.NewService(db.DB)
	requestService := request.NewService(db.DB)
	chatService := chat.NewService(db.DB)
	messageService := message.NewService(db.DB)

	mediaStore, err := media.NewFileSystemStore(cfg.UploadDir)
	if err != nil {
		slog.Error("failed to initialize media store", "error", err)
		os.Exit(1)
	}
	mediaService := media.NewService(db.DB, mediaStore, cfg.MaxUploadSize)
	groupService := group.NewService(db.DB)

	// WebSocket Hub & Router
	wsHub := ws.NewHub(db.DB)
	wsRouter := ws.NewRouter(wsHub, messageService, db.DB)
	wsHandler := ws.NewHandler(wsHub, wsRouter, authService)

	// Handlers
	authHandler := auth.NewHandler(authService)
	userHandler := user.NewHandler(userService)
	contactHandler := contact.NewHandler(contactService)
	requestHandler := request.NewHandler(requestService)
	chatHandler := chat.NewHandler(chatService)
	messageHandler := message.NewHandler(messageService)
	mediaHandler := media.NewHandler(mediaService)
	groupHandler := group.NewHandler(groupService)
	healthHandler := health.NewHandler()

	// 4. HTTP Router
	r := chi.NewRouter()

	// Core Middlewares
	r.Use(chiMiddleware.Recoverer)
	r.Use(middleware.Logging)
	r.Use(middleware.RateLimit(cfg.RateLimitRPM))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Routes
	r.Get("/api/v1/health", healthHandler.ServeHTTP)
	r.Handle("/ws/v1/chat", wsHandler)

	// Unauthenticated Auth Routes
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.Post("/refresh", authHandler.Refresh)
	})

	// Authenticated Routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(authService))

		r.Post("/api/v1/auth/logout", authHandler.Logout)
		r.Get("/api/v1/auth/me", authHandler.Me)

		// Users
		r.Get("/api/v1/users/search", userHandler.Search)
		r.Get("/api/v1/users/{userId}", userHandler.GetByID)
		r.Put("/api/v1/users/me", userHandler.UpdateMe)

		// Contacts
		r.Get("/api/v1/contacts", contactHandler.List)
		r.Post("/api/v1/contacts", contactHandler.Add)
		r.Delete("/api/v1/contacts/{id}", contactHandler.Remove)

		// Message Requests
		r.Get("/api/v1/message-requests", requestHandler.List)
		r.Post("/api/v1/message-requests/{id}/accept", requestHandler.Accept)
		r.Post("/api/v1/message-requests/{id}/decline", requestHandler.Decline)

		// Chats & Messages
		r.Get("/api/v1/chats", chatHandler.ListChats)
		r.Get("/api/v1/chats/{id}/messages", chatHandler.GetMessages)

		r.Post("/api/v1/messages", messageHandler.Create)
		r.Put("/api/v1/messages/{id}", messageHandler.Edit)
		r.Delete("/api/v1/messages/{id}", messageHandler.Delete)

		// Groups
		r.Post("/api/v1/groups", groupHandler.Create)
		r.Post("/api/v1/groups/{id}/members", groupHandler.AddMember)
		r.Delete("/api/v1/groups/{id}/members/{userId}", groupHandler.RemoveMember)

		// Media Upload & Serving
		r.Post("/api/v1/media/upload", mediaHandler.Upload)
		r.Get("/api/v1/media/{mediaId}", mediaHandler.Download)
	})

	// 5. Server Lifecyle
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server listener failure", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info(fmt.Sprintf("syncslate backend listening on http://localhost:%s", cfg.Port))

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	slog.Info("shutting down server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("syncslate backend server stopped cleanly")
}
