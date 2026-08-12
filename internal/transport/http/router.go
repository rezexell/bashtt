package httptransport

import (
	"log/slog"
	"net/http"
)

type Router struct {
	logger *slog.Logger
}

func NewRouter(logger *slog.Logger) *Router {
	return &Router{
		logger: logger,
	}
}

func (r *Router) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", r.health)

	return loggingMiddleware(r.logger, mux)
}

func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		logger.Info(
			"http request",
			"method", req.Method,
			"path", req.URL.Path,
		)

		next.ServeHTTP(w, req)
	})
}
