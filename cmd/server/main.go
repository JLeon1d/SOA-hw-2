package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"marketplace-backend/internal/middleware"
	"marketplace-backend/internal/operations"
	"marketplace-backend/internal/orders"
	"marketplace-backend/internal/products"
	"marketplace-backend/internal/promos"
	"marketplace-backend/internal/users"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
)

func connectWithRetry(dsn string, maxRetries int, retryDelay time.Duration) (*sqlx.DB, error) {
	var db *sqlx.DB
	var err error

	for i := 0; i < maxRetries; i++ {
		db, err = sqlx.Connect("postgres", dsn)
		if err == nil {
			return db, nil
		}

		log.Printf("Failed to connect to database (attempt %d/%d): %v", i+1, maxRetries, err)
		if i < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}

	return nil, fmt.Errorf("failed to connect after %d attempts: %w", maxRetries, err)
}

func runMigrations(logger zerolog.Logger) error {
	migrateURL := os.Getenv("DATABASE_URL")
	logger.Info().Str("url", maskPassword(migrateURL)).Msg("Running database migrations...")

	m, err := migrate.New(
		"file://migrations",
		migrateURL,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	// Run migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	if err == migrate.ErrNoChange {
		logger.Info().Msg("No new migrations to apply")
	} else {
		logger.Info().Msg("Migrations applied successfully")
	}

	return nil
}

func main() {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Parse DATABASE_URL to lib/pq format
	databaseURL := os.Getenv("DATABASE_URL")
	dsn := parseDatabaseURL(databaseURL)

	logger.Info().Str("dsn", maskPassword(dsn)).Msg("Connecting to database")

	jwtSecret := getEnv("JWT_SECRET", "your-secret-key-change-in-production")

	// Retry connection up to 10 times with 2 second delay
	db, err := connectWithRetry(dsn, 10, 2*time.Second)
	if err != nil {
		logger.Error().Err(err).Str("dsn", maskPassword(dsn)).Msg("Failed to connect to database")
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	logger.Info().Msg("Database connection established")

	// Run migrations
	if err := runMigrations(logger); err != nil {
		logger.Error().Err(err).Msg("Failed to run migrations")
		log.Fatalf("Failed to run migrations: %v", err)
	}

	usersRepo := users.NewRepository(db)
	usersService := users.NewService(usersRepo, jwtSecret, 30*time.Minute, 7*24*time.Hour)
	usersHandler := users.NewHandler(usersService)

	productsRepo := products.NewRepository(db)
	productsService := products.NewService(productsRepo)
	productsHandler := products.NewHandler(productsService)

	promosRepo := promos.NewRepository(db)
	promosService := promos.NewService(promosRepo)
	promosHandler := promos.NewHandler(promosService)

	operationsRepo := operations.NewRepository(db)

	ordersRepo := orders.NewRepository(db)
	ordersService := orders.NewService(ordersRepo, productsRepo, promosRepo, operationsRepo, 5)
	ordersHandler := orders.NewHandler(ordersService)

	r := chi.NewRouter()

	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.RequestIDMiddleware)
	r.Use(middleware.LoggingMiddleware)

	r.Post("/auth/register", usersHandler.Register)
	r.Post("/auth/login", usersHandler.Login)
	r.Post("/auth/refresh", usersHandler.Refresh)

	r.Route("/products", func(r chi.Router) {
		r.Get("/", productsHandler.ListProducts)
		r.Get("/{id}", productsHandler.GetProduct)

		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthMiddleware(usersService))
			r.Post("/", productsHandler.CreateProduct)
			r.Put("/{id}", productsHandler.UpdateProduct)
			r.Delete("/{id}", productsHandler.DeleteProduct)
		})
	})

	r.Route("/orders", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(usersService))
		r.Post("/", ordersHandler.CreateOrder)
		r.Get("/{id}", ordersHandler.GetOrder)
		r.Put("/{id}", ordersHandler.UpdateOrder)
		r.Post("/{id}/cancel", ordersHandler.CancelOrder)
	})

	r.Route("/promo-codes", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(usersService))
		r.Post("/", promosHandler.CreatePromoCode)
		r.Get("/{id}", promosHandler.GetPromoCode)
		r.Put("/{id}", promosHandler.UpdatePromoCode)
		r.Delete("/{id}", promosHandler.DeletePromoCode)
	})

	port := getEnv("PORT", "8080")
	addr := fmt.Sprintf(":%s", port)

	logger.Info().Str("addr", addr).Msg("Starting server")

	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func parseDatabaseURL(databaseURL string) string {
	// Convert postgres://user:pass@host:port/dbname?params to lib/pq format
	if len(databaseURL) > 11 && databaseURL[:11] == "postgres://" {
		rest := databaseURL[11:]
		parts := splitOnce(rest, "@")
		if len(parts) == 2 {
			userPass := parts[0]
			hostPortDb := parts[1]

			userPassParts := splitOnce(userPass, ":")
			user := userPassParts[0]
			password := ""
			if len(userPassParts) == 2 {
				password = userPassParts[1]
			}

			hostPortDbParts := splitOnce(hostPortDb, "/")
			hostPort := hostPortDbParts[0]
			dbParams := ""
			if len(hostPortDbParts) == 2 {
				dbParams = hostPortDbParts[1]
			}

			hostPortParts := splitOnce(hostPort, ":")
			host := hostPortParts[0]
			port := "5432"
			if len(hostPortParts) == 2 {
				port = hostPortParts[1]
			}

			dbParamsParts := splitOnce(dbParams, "?")
			dbname := dbParamsParts[0]

			return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
				host, port, user, password, dbname)
		}
	}
	return databaseURL
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func maskPassword(dsn string) string {
	// Simple password masking for logging
	if len(dsn) > 20 {
		return dsn[:20] + "...***..."
	}
	return "***"
}

func splitOnce(s, sep string) []string {
	idx := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			idx = i
			return []string{s[:idx], s[idx+len(sep):]}
		}
	}
	return []string{s}
}
