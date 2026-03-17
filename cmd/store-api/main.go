// Package main arma la aplicacion completa y levanta el servidor HTTP.
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"tienda-go/internal/api"
	"tienda-go/internal/bootstrap"
	"tienda-go/internal/config"
	"tienda-go/internal/partnerusers"
	"tienda-go/internal/partnerusers/mysqlrepo"
	"tienda-go/internal/partnerusers/partnerapi"
	"tienda-go/internal/service"
	"tienda-go/internal/store"
)

// main carga configuracion, crea dependencias, siembra datos y arranca la API.
func main() {
	cfg := config.Load()
	// slog queda como logger estructurado nativo, suficiente para este proyecto.
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Aqui se cablean todas las dependencias manualmente para que se vea claro
	// como se relacionan las capas sin introducir un contenedor DI.
	appStore, cleanup, err := openStore(cfg)
	if err != nil {
		logger.Error("unable to initialize persistence", slog.Any("error", err))
		os.Exit(1)
	}
	defer cleanup()

	userService := service.NewUserService(appStore)
	authService := service.NewAuthService(appStore, cfg.TokenTTL)
	productService := service.NewProductService(appStore)
	inventoryService := service.NewInventoryService(appStore)
	orderService := service.NewOrderService(appStore)
	dashboardService := service.NewDashboardService(appStore)

	seedData := bootstrap.SeedData{}
	if cfg.SeedOnStart {
		seedData, err = bootstrap.Seed(userService, productService)
		if err != nil {
			logger.Error("unable to seed application", slog.Any("error", err))
			os.Exit(1)
		}
	}

	httpAPI := api.New(
		cfg,
		logger,
		authService,
		userService,
		productService,
		inventoryService,
		orderService,
		dashboardService,
	)
	partnerCleanup := func() {}
	if partnerRoutes, cleanup, err := buildPartnerUsersRoutes(cfg, httpAPI); err != nil {
		logger.Warn("partner users routes disabled", slog.Any("error", err))
	} else if partnerRoutes != nil {
		httpAPI.SetPartnerRoutes(partnerRoutes)
		partnerCleanup = cleanup
		logger.Info("partner users routes enabled")
	}
	defer partnerCleanup()

	app := httpAPI.App()

	logger.Info("store API ready",
		slog.String("address", cfg.Address),
		slog.String("environment", cfg.Env),
		slog.String("storage", cfg.StoreDriver),
	)

	for _, user := range seedData.Users {
		// Mostrar credenciales demo simplifica la exploracion inicial del proyecto.
		logger.Info("seed user",
			slog.String("role", string(user.Role)),
			slog.String("email", user.Email),
			slog.String("password", user.Password),
		)
	}

	serverErrors := make(chan error, 1)
	go func() {
		// Listen bloquea, por eso se ejecuta en una goroutine.
		serverErrors <- app.Listen(cfg.Address)
	}()

	signalContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErrors:
		if err != nil {
			logger.Error("server stopped unexpectedly", slog.Any("error", err))
			os.Exit(1)
		}
		return
	case <-signalContext.Done():
	}

	// Cuando llega una senal, se intenta apagar el servidor sin cortar requests activas.
	shutdownContext, cancel := context.WithTimeout(context.Background(), cfg.ShutdownWait)
	defer cancel()

	logger.Info("shutting down server")
	if err := app.ShutdownWithContext(shutdownContext); err != nil {
		logger.Error("graceful shutdown failed", slog.Any("error", err))
	}

	select {
	case err := <-serverErrors:
		if err != nil {
			normalized := strings.ToLower(err.Error())
			if !strings.Contains(normalized, "server closed") && !strings.Contains(normalized, "closed network connection") {
				logger.Error("server stopped after shutdown", slog.Any("error", err))
			}
		}
	case <-time.After(2 * time.Second):
	}
}

func openStore(cfg config.Config) (store.Store, func(), error) {
	driver := strings.ToLower(strings.TrimSpace(cfg.StoreDriver))
	switch driver {
	case "", "memory":
		return store.NewMemoryStore(), func() {}, nil
	case "mysql":
		mysqlStore, err := store.NewMySQLStore(cfg.MySQLDSN, cfg.MySQLAutoMigrate)
		if err != nil {
			return nil, nil, err
		}

		return mysqlStore, func() {
			if err := mysqlStore.Close(); err != nil {
				slog.Error("failed to close mysql store", slog.Any("error", err))
			}
		}, nil
	default:
		return nil, nil, errors.New("unsupported store driver: " + driver)
	}
}

func buildPartnerUsersRoutes(cfg config.Config, httpAPI *api.API) (func(router fiber.Router), func(), error) {
	if strings.TrimSpace(cfg.PartnerUsersProdDSN) == "" {
		return nil, func() {}, nil
	}

	cleanups := make([]func(), 0, 2)

	prodRepos, err := mysqlrepo.OpenPartnerOnly(mysqlrepo.Config{
		PartnerDSN: cfg.PartnerUsersProdDSN,
	})
	if err != nil {
		return nil, nil, err
	}
	cleanups = append(cleanups, func() {
		if err := prodRepos.Close(); err != nil {
			slog.Error("failed to close partner users prod repositories", slog.Any("error", err))
		}
	})

	prodOnlyService := partnerusers.NewService(
		nil,
		nil,
		prodRepos.Subscribers,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		"",
	)
	prodOnlyHandler := partnerusers.NewHandler(prodOnlyService, httpAPI.PartnerUsersCurrentUser)
	prodOnlyRoutes := func(router fiber.Router) {
		prodOnlyHandler.RegisterProdOnlyRoutes(router)
	}
	slog.Info("partner users prod-only routes enabled")

	if strings.TrimSpace(cfg.PartnerUsersAppDSN) == "" ||
		strings.TrimSpace(cfg.PartnerAPIUser) == "" ||
		strings.TrimSpace(cfg.PartnerAPIPass) == "" ||
		strings.TrimSpace(cfg.PartnerAPIBaseURL) == "" {
		return prodOnlyRoutes, func() {
			for _, cleanup := range cleanups {
				cleanup()
			}
		}, nil
	}

	repos, err := mysqlrepo.Open(mysqlrepo.Config{
		AppDSN:     cfg.PartnerUsersAppDSN,
		PartnerDSN: cfg.PartnerUsersProdDSN,
		UsersTable: "Users",
	})
	if err != nil {
		slog.Warn("partner users full routes disabled", slog.Any("error", err))
		return prodOnlyRoutes, func() {
			for _, cleanup := range cleanups {
				cleanup()
			}
		}, nil
	}
	cleanups = append(cleanups, func() {
		if err := repos.Close(); err != nil {
			slog.Error("failed to close partner users repositories", slog.Any("error", err))
		}
	})

	apiClient, err := partnerapi.New(partnerapi.Config{
		BaseURL:       cfg.PartnerAPIBaseURL,
		Partner:       cfg.PartnerAPIUser,
		Password:      cfg.PartnerAPIPass,
		Timeout:       cfg.PartnerAPITimeout,
		SkipTLSVerify: cfg.PartnerAPISkipTLSVerify,
	})
	if err != nil {
		slog.Warn("partner users full routes disabled", slog.Any("error", err))
		return prodOnlyRoutes, func() {
			for _, cleanup := range cleanups {
				cleanup()
			}
		}, nil
	}

	service := partnerusers.NewService(
		repos.Users,
		nil,
		repos.Subscribers,
		apiClient,
		repos.Audits,
		partnerusers.ErrorScheduler{},
		nil,
		partnerusers.DefaultPackageCatalog{},
		partnerusers.DefaultEarlyDeactivationPolicy{},
		partnerusers.DefaultVigenciaPolicy{},
		cfg.PartnerLoginBaseURL,
	)

	handler := partnerusers.NewHandler(service, httpAPI.PartnerUsersCurrentUser)

	return func(router fiber.Router) {
			prodOnlyHandler.RegisterProdOnlyRoutes(router)
			handler.RegisterRoutes(router)
		}, func() {
			for _, cleanup := range cleanups {
				cleanup()
			}
		}, nil
}
