package main

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"
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

	logger.Log.Info("Configuration",
		zap.String("ras_addr", rasAddr),
		zap.String("bind_addr", bindAddr),
	)

	// Создание сервера
	server := ras.NewRASServer(rasAddr)

	// Канал для ошибок сервера
	serverErrors := make(chan error, 1)

	// Запуск сервера в горутине
	go func() {
		logger.Log.Info("Starting gRPC server", zap.String("address", bindAddr))
		if err := server.Serve(bindAddr); err != nil {
			serverErrors <- err
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

		// Остановка сервера
		logger.Log.Info("Shutting down server gracefully...")
		if err := server.GracefulStop(ctx); err != nil {
			logger.Log.Error("Error during shutdown", zap.Error(err))
			return err
		}

		logger.Log.Info("Server stopped successfully")
	}

	return nil
}
