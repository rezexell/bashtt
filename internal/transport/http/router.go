package httptransport

import (
	"log/slog"
	"net/http"
	"time"
)

type Router struct {
	logger *slog.Logger

	createHandler   *CreateHandler
	callbackHandler *CallbackHandler
	healthHandler   *HealthHandler
}

func NewRouter(
	logger *slog.Logger,
	createHandler *CreateHandler,
	callbackHandler *CallbackHandler,
	healthHandler *HealthHandler,
) *Router {
	return &Router{
		logger:          logger,
		createHandler:   createHandler,
		callbackHandler: callbackHandler,
		healthHandler:   healthHandler,
	}
}

func (r *Router) CreateHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc(
		"GET /health",
		r.healthHandler.Health,
	)

	mux.HandleFunc(
		"POST /create",
		r.createHandler.Create,
	)

	return loggingMiddleware(
		r.logger,
		mux,
	)
}

func (r *Router) CallbackHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc(
		"GET /health",
		r.healthHandler.Health,
	)

	mux.HandleFunc(
		"POST /callback",
		r.callbackHandler.Callback,
	)

	return loggingMiddleware(
		r.logger,
		mux,
	)
}

func loggingMiddleware(
	logger *slog.Logger,
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			next.ServeHTTP(w, r)

			logger.Info(
				"http request",
				"method", r.Method,
				"path", r.URL.Path,
				"duration", time.Since(start),
			)
		},
	)
}
