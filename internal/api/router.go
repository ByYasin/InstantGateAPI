package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/proyaai/instantgate/internal/api/handlers"
	mw "github.com/proyaai/instantgate/internal/api/middleware"
	"github.com/proyaai/instantgate/internal/cache"
	"github.com/proyaai/instantgate/internal/config"
	"github.com/proyaai/instantgate/internal/database/mysql"
	"github.com/proyaai/instantgate/internal/security"
	"github.com/proyaai/instantgate/internal/validation"
)

type Server struct {
	config            *config.Config
	router            *chi.Mux
	introspector      *mysql.Introspector
	schemaCache       *mysql.SchemaCache
	jwtManager        *security.JWTManager
	accessControl     *security.AccessControl
	validationManager *validation.ValidationManager
	cache             *cache.Cache
	healthHandler     *handlers.HealthHandler
	schemaHandler     *handlers.SchemaHandler
	genericHandler    *handlers.GenericHandler
	httpServer        *http.Server
}

func NewServer(cfg *config.Config) (*Server, error) {
	s := &Server{
		config:        cfg,
		router:        chi.NewRouter(),
		accessControl: security.NewAccessControl(&cfg.Security),
		jwtManager:    security.NewJWTManager(&cfg.JWT),
	}

	s.introspector = mysql.NewIntrospector(&cfg.Database)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.introspector.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	schemaCache, err := s.introspector.LoadSchema(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load schema: %w", err)
	}
	s.schemaCache = schemaCache

	if cfg.Redis.Host != "" {
		cache, err := cache.NewCache(&cfg.Redis)
		if err != nil {
			fmt.Printf("Warning: failed to initialize Redis cache: %v\n", err)
		} else {
			s.cache = cache
		}
	}

	s.healthHandler = handlers.NewHealthHandler(s.introspector.GetDB())
	s.schemaHandler = handlers.NewSchemaHandler(s.schemaCache)
	s.validationManager = validation.NewValidationManager(&cfg.Validation, s.schemaCache)
	s.genericHandler = handlers.NewGenericHandler(s.introspector.GetDB(), s.schemaCache, s.validationManager)

	s.setupRoutes()

	return s, nil
}

func (s *Server) setupRoutes() {
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.Timeout(60 * time.Second))

	s.router.Use(mw.Recovery())
	s.router.Use(mw.Logger())
	s.router.Use(mw.RequestID())

	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"X-Total-Count", "X-Limit", "X-Offset", "X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           300,
	})
	s.router.Use(corsMiddleware.Handler)

	s.router.Get("/health", s.healthHandler.Check)

	apiRouter := chi.NewRouter()

	if s.config.Security.RequireAuth {
		apiRouter.Use(mw.JWTAuth(s.jwtManager))
	} else {
		apiRouter.Use(mw.OptionalJWTAuth(s.jwtManager))
	}

	apiRouter.Get("/schema", s.schemaHandler.ListTables)
	apiRouter.Get("/schema/{table}", s.schemaHandler.GetTableSchema)

	crudGroup := apiRouter.With(mw.TableAccessControl(s.accessControl))

	crudGroup.Get("/{table}", s.genericHandler.ListTable)
	crudGroup.Get("/{table}/{id}", s.genericHandler.GetByID)
	crudGroup.Post("/{table}", s.genericHandler.Create)
	crudGroup.Patch("/{table}/{id}", s.genericHandler.Update)
	crudGroup.Delete("/{table}/{id}", s.genericHandler.Delete)

	s.router.Mount("/api", apiRouter)

	s.router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		handlers.SendJSON(w, r, http.StatusOK, map[string]interface{}{
			"name":        "InstantGate API",
			"version":     "1.0.0",
			"description": "Instant REST API for any relational database",
		})
	})
}

func (s *Server) Start() error {
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.Server.Port),
		Handler:      s.router,
		ReadTimeout:  s.config.Server.ReadTimeout,
		WriteTimeout: s.config.Server.WriteTimeout,
		IdleTimeout:  s.config.Server.IdleTimeout,
	}

	fmt.Printf("Starting InstantGate API on port %d\n", s.config.Server.Port)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	var errs []error

	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("HTTP server shutdown: %w", err))
		}
	}

	if s.introspector != nil {
		if err := s.introspector.Close(); err != nil {
			errs = append(errs, fmt.Errorf("database close: %w", err))
		}
	}

	if s.cache != nil {
		if err := s.cache.Close(); err != nil {
			errs = append(errs, fmt.Errorf("cache close: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}

	return nil
}

func (s *Server) GetRouter() *chi.Mux {
	return s.router
}

func (s *Server) GetDB() *mysql.Introspector {
	return s.introspector
}

func (s *Server) GetConfig() *config.Config {
	return s.config
}