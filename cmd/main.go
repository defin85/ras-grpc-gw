package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/v8platform/ras-grpc-gw/pkg/health"
	"github.com/v8platform/ras-grpc-gw/pkg/logger"
	ras "github.com/v8platform/ras-grpc-gw/pkg/server"
	"go.uber.org/zap"
)

// nolint: gochecknoglobals
var (
	version = "dev"
	commit  = ""
	date    = ""
	builtBy = ""
)

func main() {
	// Инициализация logger
	debug := os.Getenv("DEBUG") == "true"
	if err := logger.Init(debug); err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	logger.Log.Info("Starting ras-grpc-gw",
		zap.String("version", version),
		zap.String("commit", commit),
		zap.String("date", date),
		zap.String("built_by", builtBy),
		zap.String("go_version", runtime.Version()),
	)

	app := &cli.App{
		Name:    "ras-grpc-gw",
		Version: version,
		Authors: []*cli.Author{
			{
				Name: "Aleksey Khorev",
			},
		},
		UsageText:   "ras-grpc-wg [OPTIONS] [HOST:PORT]",
		Copyright:   "(c) 2021 Khorevaa",
		Description: "GRPC gateway for RAS 1S.Enterprise",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "bind",
				Value: ":3002",
				Usage: "host:port to bind grpc server",
			},
			&cli.StringFlag{
				Name:    "health",
				Value:   "0.0.0.0:8080",
				Usage:   "HTTP health check server address",
				EnvVars: []string{"HEALTH_ADDR"},
			},
		},
		Action: runServer,
	}

	if err := app.Run(os.Args); err != nil {
		logger.Log.Fatal("Application failed", zap.Error(err))
	}
}

func runServer(c *cli.Context) error {
	rasAddr := "localhost:1545"
	if c.Args().Present() {
		rasAddr = c.Args().First()
	}

	bindAddr := c.String("bind")
	healthAddr := c.String("health")

	logger.Log.Info("Configuration",
		zap.String("ras_addr", rasAddr),
		zap.String("bind_addr", bindAddr),
		zap.String("health_addr", healthAddr),
	)

	// Создание gRPC сервера
	server := ras.NewRASServer(rasAddr)

	// Создание HTTP health check сервера
	healthSrv := health.NewServer(healthAddr, server)

	// Канал для ошибок серверов
	serverErrors := make(chan error, 2)

	// Запуск gRPC сервера
	go func() {
		logger.Log.Info("Starting gRPC server", zap.String("address", bindAddr))
		if err := server.Serve(bindAddr); err != nil {
			serverErrors <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	// Запуск HTTP health check сервера
	go func() {
		if err := healthSrv.Start(); err != nil && err != http.ErrServerClosed {
			serverErrors <- fmt.Errorf("health server error: %w", err)
		}
	}()

	// Канал для системных сигналов
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Ожидание сигнала или ошибки
	select {
	case err := <-serverErrors:
		return err
	case sig := <-shutdown:
		logger.Log.Info("Shutdown signal received",
			zap.String("signal", sig.String()),
		)

		// Graceful shutdown с таймаутом 30 секунд
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Остановка обоих серверов
		logger.Log.Info("Shutting down servers gracefully...")

		// Остановка health сервера
		if err := healthSrv.Shutdown(ctx); err != nil {
			logger.Log.Error("Error shutting down health server", zap.Error(err))
		}

		// Остановка gRPC сервера
		if err := server.GracefulStop(ctx); err != nil {
			logger.Log.Error("Error shutting down gRPC server", zap.Error(err))
			return err
		}

		logger.Log.Info("Servers stopped successfully")
	}

	return nil
}
