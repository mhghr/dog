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

	"monitoring-platform/internal/agents"
	"monitoring-platform/internal/alerting"
	"monitoring-platform/internal/auth"
	"monitoring-platform/internal/config"
	"monitoring-platform/internal/events"
	"monitoring-platform/internal/health"
	"monitoring-platform/internal/ingestion"
	"monitoring-platform/internal/metrics"
	"monitoring-platform/internal/postgres"
	"monitoring-platform/internal/queue"
	"monitoring-platform/internal/repository"
)

type Deps struct {
	Config         *config.Config
	Logger         *slog.Logger
	Monitors       repository.MonitorRepository
	Results        repository.ResultRepository
	Locations      repository.LocationRepository
	StatusPages    repository.StatusPageRepository
	Orgs           repository.OrganizationRepository
	Projects       repository.ProjectRepository
	AlertRepo      *postgres.AlertRepository
	ChannelRepo    *postgres.ChannelRepository
	AlertEngine    *alerting.Engine
	Notifier       *alerting.Notifier
	HealthRepo     *postgres.HealthRepository
	HealthNotifier *health.NotificationEngine
	Ingestion      *ingestion.Service
	Auth           *auth.Service
	Issuer         *auth.TokenIssuer
	Bus            *events.Bus
	Queue          *queue.RedisQueue
	Pool           *pgxpool.Pool
	Redis          *redis.Client
	Victoria       *metrics.VictoriaClient
	Prom           http.Handler
	AgentRepo      *agents.Repository
	CA             *agents.CertAuthority
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

	orgScoped := auth.OrgScoped(deps.Issuer)

	requireAdmin := auth.RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, r, http.StatusForbidden, "forbidden", "Admin access required", nil)
	})

	router.Get("/health/live", handler.healthLive)
	router.Get("/health/ready", handler.healthReady)
	router.Method(http.MethodGet, "/metrics", deps.Prom)

	router.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.Timeout(60 * time.Second))

		// Public, unauthenticated status page projection.
		r.Get("/status-pages/public/{slug}", handler.publicStatusPage)

		// Agent enrollment (public, token-authenticated)
		r.Post("/agent/v1/enroll", handler.agentEnroll)

		// Agent status polling (public, agent ID lookup only)
		r.Get("/agent/v1/status/{agentID}", handler.agentStatus)

		// Public auth
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

			r.Route("/organizations", func(r chi.Router) {
				r.Post("/", handler.createOrganization)

				r.Group(func(r chi.Router) {
					r.Use(orgScoped)
					r.Get("/projects", handler.listProjects)
					r.Post("/projects", handler.createProject)
				})
			})

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

					r.Route("/parameter-rules", func(r chi.Router) {
						r.Get("/", handler.listParameterRules)
						r.Route("/{parameterKey}", func(r chi.Router) {
							r.Get("/", handler.getParameterRule)
							r.Put("/", handler.putParameterRule)
							r.Delete("/", handler.deleteParameterRule)
						})
					})

					r.Route("/notification-policies", func(r chi.Router) {
						r.Get("/", handler.listNotificationPolicies)
						r.Post("/", handler.createNotificationPolicy)
					})
				})
			})

			r.Get("/monitor-types/{type}/parameters", handler.listMonitorTypeParameters)

			r.Route("/notification-channels", func(r chi.Router) {
				r.Get("/", handler.listHealthNotificationChannels)
				r.Post("/", handler.createHealthNotificationChannel)

				r.Route("/{channelId}", func(r chi.Router) {
					r.Put("/", handler.updateHealthNotificationChannel)
					r.Delete("/", handler.deleteHealthNotificationChannel)
					r.Post("/test", handler.testHealthNotificationChannel)
				})
			})

			r.Route("/notification-policies", func(r chi.Router) {
				r.Route("/{policyId}", func(r chi.Router) {
					r.Put("/", handler.updateNotificationPolicy)
					r.Delete("/", handler.deleteNotificationPolicy)
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

			r.Route("/alerting", func(r chi.Router) {
				r.Route("/policies", func(r chi.Router) {
					r.Get("/", handler.listAlertPolicies)
					r.Post("/", handler.createAlertPolicy)
				})

				r.Route("/channels", func(r chi.Router) {
					r.Get("/", handler.listNotificationChannels)
					r.Post("/", handler.createNotificationChannel)
				})

				r.Get("/alerts", handler.listAlerts)
				r.Get("/alerts/{alertID}", handler.getAlert)
			})

			r.Route("/admin", func(r chi.Router) {
				r.Use(requireAdmin)

				r.Route("/probe-agents", func(r chi.Router) {
					r.Get("/", handler.listAgents)
					r.Get("/{agentID}", handler.getAgent)
					r.Post("/{agentID}/approve", handler.approveAgent)
					r.Post("/{agentID}/reject", handler.rejectAgent)
					r.Post("/{agentID}/disable", handler.disableAgent)
					r.Post("/{agentID}/enable", handler.enableAgent)
					r.Post("/{agentID}/revoke", handler.revokeAgent)
					r.Post("/{agentID}/drain", handler.drainAgent)
				})

				r.Get("/probe-agent-enrollment-tokens", handler.listEnrollmentTokens)
				r.Post("/probe-agent-enrollment-tokens", handler.createEnrollmentToken)
				r.Put("/probe-agents/{agentID}/public-ip", handler.updateAgentPublicIP)
				r.Put("/probe-agents/{agentID}/location", handler.updateAgentLocation)
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
