package server

import (
	"codek7/common/pb"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/lumbrjx/codek7/gateway/internal/api"
	"github.com/lumbrjx/codek7/gateway/internal/infra"
	"github.com/lumbrjx/codek7/gateway/internal/middlewares"
	"github.com/lumbrjx/codek7/gateway/internal/watcher"
	// "github.com/lai0xn/codek-gateway/internal/middlewares"
)

type Server struct {
	router  *chi.Mux
	port    string
	api     *api.API
	watcher *watcher.Watcher
	hub     *watcher.Hub
}

// NewServer creates a new server instance
func NewServer(port string) *Server {
	if port == "" {
		port = "8080"
	}

	kafkaProducer, err := infra.MakeKafkaProducer()
	if err != nil {
		log.Fatalf("Error: %v", err)
		os.Exit(1)
	}
	grpcClient := pb.NewRepoServiceClient(
		infra.MakeGRPCClientConn(),
	)

	// Initialize WebSocket hub
	hub := watcher.NewHub()

	// Initialize watcher
	watcherInstance, err := watcher.NewWatcher(hub)
	if err != nil {
		log.Fatalf("Failed to create watcher: %v", err)
	}

	s := &Server{
		router:  chi.NewRouter(),
		port:    port,
		api:     &api.API{Producer: kafkaProducer, RepoClient: grpcClient, Hub: hub},
		watcher: watcherInstance,
		hub:     hub,
	}

	s.setupMiddleware()
	s.setupRoutes()

	// Start the hub in a goroutine
	go hub.Run()

	// Start the watcher in a goroutine
	if err := watcherInstance.Start(); err != nil {
		log.Fatalf("Failed to start watcher: %v", err)
	}

	return s
}

// setupMiddleware configures Chi middleware
func (s *Server) setupMiddleware() {
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.SetHeader("Content-Type", "application/json"))
	s.router.Use(middleware.Timeout(60 * time.Second))
	// s.router.Use(middlewares.RateLimitMiddleware(infra.GetRDB(), 10, time.Minute*1))
	// CORS middleware
	s.router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")
			w.Header().Set("Access-Control-Allow-Credentials", "true") // ✅ correct

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})
}

// setupRoutes configures all routes
func (s *Server) setupRoutes() {
	// Health check endpoint
	s.router.Get("/health", s.api.HealthCheck)
	// Videos routes group
	s.router.Route("/videos", func(r chi.Router) {
		r.Use(middlewares.AuthMiddleware)

		r.Post("/upload", s.api.UploadFile)
		r.Get("/{video_id}/download", s.api.DownloadVideo)
		r.Get("/{video_id}", s.api.GetVideoByID)
		r.Get("/user/{user_id}", s.api.GetUserVideos)
		r.Get("/recent/{user_id}", s.api.GetRecentUserVideos)
	})
	// Static and streaming routes with auth
	s.router.Route("/static", func(r chi.Router) {
		r.Use(middlewares.AuthMiddleware)
		r.Handle("/*", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	})

	s.router.Route("/hls", func(r chi.Router) {
		r.Use(middlewares.AuthMiddleware)
		r.Get("/*", s.api.StreamFromMinIO)
	})

	// WebSocket endpoint for notifications with auth
	s.router.Route("/ws", func(r chi.Router) {
		r.Use(middlewares.AuthMiddleware)
		r.Get("/notifications", s.api.WebSocketHandler)
	})

	s.router.Get("/er/:user_id" , s.api.ErHandler)

	// Auth routes
	s.router.Post("/auth/login", s.api.Login)
	s.router.Post("/auth/logout", s.api.Logout)
	s.router.Post("/auth/register", s.api.Register)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	srv := &http.Server{
		Addr:         ":" + s.port,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Server starting on port %s", s.port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Server shutting down...")

	// Close watcher
	if s.watcher != nil {
		if err := s.watcher.Close(); err != nil {
			log.Printf("Error closing watcher: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
		return err
	}

	log.Println("Server stopped")
	return nil
}
