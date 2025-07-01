package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/lai0xn/codek-gateway/internal/api"
	"github.com/lai0xn/codek-gateway/internal/infra"
	// "github.com/lai0xn/codek-gateway/internal/middlewares"
)

type Server struct {
	router *chi.Mux
	port   string
	api    *api.API
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

	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	s := &Server{
		router: chi.NewRouter(),
		port:   port,
		api:    &api.API{Producer: kafkaProducer},
	}

	s.setupMiddleware()
	s.setupRoutes()

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
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")

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
	s.router.Post("/videos/upload", s.api.UploadFile)
	s.router.Get("/videos/{video_id}/download", s.api.DownloadVideo)
	s.router.Get("/videos/{video_id}", s.api.GetVideoByID)
	s.router.Get("/videos/user/{user_id}", s.api.GetUserVideos)
	s.router.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	s.router.Get("/hls/*", s.api.StreamFromMinIO)
	// Auth routes
	// s.router.Post("/auth/login", s.api.LoginHandler)
	// s.router.Post("/auth/logout", s.api.LogoutHandler)
	// s.router.Post("/auth/register", s.api.RegisterHandler)
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
		return err
	}

	log.Println("Server stopped")
	return nil
}
