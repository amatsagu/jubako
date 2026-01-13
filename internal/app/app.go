package app

import (
	"context"
	"jubako/internal/config"
	"jubako/internal/swarm"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/amatsagu/lumo"
)

type App struct {
	Ctx       context.Context
	CancelCtx context.CancelFunc
	StartedAt *time.Time // In local time

	SwarmClient *swarm.SwarmClient
	HttpServer  *http.Server
}

func NewApplication() *App {
	lumo.Debug("Creating main application instance.")

	ctx, cancel := context.WithCancel(context.Background())
	mux := http.NewServeMux()

	sc := swarm.NewSwarmClient()
	mux.HandleFunc("/stream", sc.StreamHandler)

	app := &App{
		Ctx:         ctx,
		CancelCtx:   cancel,
		SwarmClient: sc,
		HttpServer: &http.Server{
			Addr:    "127.0.0.1:" + config.HTTP_PORT,
			Handler: mux,
		},
	}

	app.HttpServer.SetKeepAlivesEnabled(true)
	return app
}

func (app *App) Run() {
	startedAt := time.Now()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	serverErr := make(chan error, 1)
	app.StartedAt = &startedAt

	lumo.Info("Started HTTP server at 127.0.0.1:%s!", config.HTTP_PORT)
	go func() {
		serverErr <- app.HttpServer.ListenAndServe()
	}()

	select {
	case <-app.Ctx.Done():
		lumo.Info("Received core app context cancellation -> requested to shut down app handler...")
	case sig := <-stop:
		lumo.Info("Received \"%v\" signal -> requested to shut down app handler...", sig)
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			werr := lumo.WrapError(err)
			lumo.Error("Received error from app's internal http server: %v", werr)
		}
	}

	app.Shutdown()
}

func (app *App) Shutdown() {
	lumo.Debug("Stopping all services watching main context...")
	app.CancelCtx()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	lumo.Debug("Stopping HTTP server...")
	if err := app.HttpServer.Shutdown(shutdownCtx); err != nil {
		lumo.Warn("Graceful shutdown failed: %v. Force closing HTTP server...\n", err)
		_ = app.HttpServer.Close()
	}

	lumo.Info("Finished shutdown process for all services. Bye!")
}
