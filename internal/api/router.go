package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"monitoring-platform/internal/auth"
	"monitoring-platform/internal/config"
	"monitoring-platform/internal/events"
	"monitoring-platform/internal/ingestion"
	"monitoring-platform/internal/metrics"
	"monitoring-platform/internal/queue"
	"monitoring-platform/internal/repository"
)

type Deps struct {
	Config      *config.Config
	Logger      *slog.Logger
	Monitors    repository.MonitorRepository
	Results     repository.ResultRepository
	Locations   repository.LocationRepository
	StatusPages repository.StatusPageRepository
	Ingestion   *ingestion.Service
	Auth        *auth.Service
	Issuer      *auth.TokenIssuer
	Bus         *events.Bus
	Queue       *queue.RedisQueue
	Pool        *pgxpool.Pool
	Redis       *redis.Client
	Victoria    *metrics.VictoriaClient
	Prom        http.Handler
}

func NewRouter(deps Deps) http.Handler {
	handler := &Handler{deps: deps}

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(requestLogger(deps.Logger))
	router.Use(middleware.Recoverer)

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins: deps.Config.CORSAllowedOrigins,
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	requireAuth := auth.RequireAuth(deps.Issuer, func(w http.ResponseWriter, r *http.Request) {
		writeError(w, r, http.StatusUnauthorized, "unauthorized", "Authentication required", nil)
	})

	router.Get("/health/live", handler.healthLive)
	router.Get("/health/ready", handler.healthReady)
	router.Method(http.MethodGet, "/metrics", deps.Prom)

	router.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.Timeout(60 * time.Second))

		// Public, unauthenticated status page projection.
		r.Get("/status-pages/public/{slug}", handler.publicStatusPage)

		r.Route("/auth", func(r chi.Router) {
			r.Post("/google/exchange", handler.googleExchange)
			r.Post("/google/mobile", handler.googleMobile)
			r.Post("/otp/request", handler.otpRequest)
			r.Post("/otp/verify", handler.otpVerify)
			r.Post("/refresh", handler.authRefresh)
			r.Post("/logout", handler.authLogout)

			r.Group(func(r chi.Router) {
				r.Use(requireAuth)
				r.Get("/me", handler.authMe)
			})
		})

		r.Group(func(r chi.Router) {
			r.Use(requireAuth)

			r.Get("/dashboard/summary", handler.dashboardSummary)
			r.Get("/probe-locations", handler.listLocations)
			r.Post("/probe-locations", handler.createLocation)
			r.Get("/system/health", handler.systemHealth)

			r.Route("/monitors", func(r chi.Router) {
				r.Get("/", handler.listMonitors)
				r.Post("/", handler.createMonitor)

				r.Route("/{monitorID}", func(r chi.Router) {
					r.Get("/", handler.getMonitor)
					r.Put("/", handler.updateMonitor)
					r.Delete("/", handler.deleteMonitor)
					r.Post("/pause", handler.pauseMonitor)
					r.Post("/resume", handler.resumeMonitor)
					r.Get("/results", handler.listMonitorResults)
					r.Get("/metrics", handler.monitorMetrics)
				})
			})

			r.Route("/status-pages", func(r chi.Router) {
				r.Get("/", handler.listStatusPages)
				r.Post("/", handler.createStatusPage)

				r.Route("/{statusPageID}", func(r chi.Router) {
					r.Get("/", handler.getStatusPage)
					r.Put("/", handler.updateStatusPage)
					r.Delete("/", handler.deleteStatusPage)
				})
			})
		})
	})

	router.Group(func(r chi.Router) {
		r.Use(requireAuth)
		r.Get("/events/v1/stream", handler.eventStream)
	})

	router.Route("/internal/v1", func(r chi.Router) {
		r.Use(handler.workerAuth)
		r.Post("/results", handler.ingestResult)
	})

	return router
}

type Handler struct {
	deps Deps
}

// requestLogger emits one structured log line per request without logging
// sensitive headers or bodies.
func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health/live" || r.URL.Path == "/health/ready" || r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			wrapped := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(wrapped, r)

			logger.Info(
				"http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.Status(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()),
				"remote_ip", r.RemoteAddr,
			)
		})
	}
}
