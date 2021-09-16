package http

import (
	"context"
	"net/http"
	"os"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"demodesk/neko/internal/config"
	"demodesk/neko/internal/types"
)

type HttpManagerCtx struct {
	logger zerolog.Logger
	config *config.Server
	router *chi.Mux
	http   *http.Server
}

func New(WebSocketManager types.WebSocketManager, ApiManager types.ApiManager, config *config.Server) *HttpManagerCtx {
	logger := log.With().Str("module", "http").Logger()

	router := chi.NewRouter()
	router.Use(middleware.Recoverer) // Recover from panics without crashing server
	router.Use(cors.Handler(cors.Options{
		AllowOriginFunc: func(r *http.Request, origin string) bool {
			return config.AllowOrigin(origin)
		},
		AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))
	router.Use(middleware.RequestID) // Create a request ID for each request
	router.Use(Logger)               // Log API request calls using custom logger function

	router.Route("/api", ApiManager.Route)

	router.Get("/api/ws", func(w http.ResponseWriter, r *http.Request) {
		WebSocketManager.Upgrade(w, r, func(r *http.Request) bool {
			return config.AllowOrigin(r.Header.Get("Origin"))
		})
	})

	if config.Static != "" {
		fs := http.FileServer(http.Dir(config.Static))
		router.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			if _, err := os.Stat(config.Static + r.URL.Path); !os.IsNotExist(err) {
				fs.ServeHTTP(w, r)
			} else {
				http.NotFound(w, r)
			}
		})
	}

	router.NotFound(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))

	return &HttpManagerCtx{
		logger: logger,
		config: config,
		router: router,
		http: &http.Server{
			Addr:    config.Bind,
			Handler: router,
		},
	}
}

func (manager *HttpManagerCtx) Start() {
	if manager.config.Cert != "" && manager.config.Key != "" {
		go func() {
			if err := manager.http.ListenAndServeTLS(manager.config.Cert, manager.config.Key); err != http.ErrServerClosed {
				manager.logger.Panic().Err(err).Msg("unable to start https server")
			}
		}()
		manager.logger.Info().Msgf("https listening on %s", manager.http.Addr)
	} else {
		go func() {
			if err := manager.http.ListenAndServe(); err != http.ErrServerClosed {
				manager.logger.Panic().Err(err).Msg("unable to start http server")
			}
		}()
		manager.logger.Info().Msgf("http listening on %s", manager.http.Addr)
	}
}

func (manager *HttpManagerCtx) Shutdown() error {
	manager.logger.Info().Msg("shutdown")

	return manager.http.Shutdown(context.Background())
}
