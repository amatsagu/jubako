package app

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"jubako/internal/config"
	"jubako/internal/route"
	"jubako/internal/swarm"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/amatsagu/lumo"
	webview "github.com/webview/webview_go"
	_ "modernc.org/sqlite"
)

type App struct {
	Ctx       context.Context
	CancelCtx context.CancelFunc
	StartedAt *time.Time // In local time

	SwarmClient *swarm.SwarmClient
	HttpServer  *http.Server
	DB          *sql.DB
	// WebView is stored directly as an interface (not a pointer to interface)
	WebView webview.WebView
}

func NewApplication(embeddedFrontend embed.FS) *App {
	lumo.Debug("Creating main application instance.")

	ctx, cancel := context.WithCancel(context.Background())
	mux := http.NewServeMux()

	db, err := sql.Open("sqlite", config.APP_FILES_PATH+"/data.db")
	if err != nil {
		werr := lumo.WrapError(err).Include("sqlite_file_path", config.APP_FILES_PATH+"/data.db")
		lumo.Panic("Failed to open local sqlite database: %v", werr)
	}

	sc := swarm.NewSwarmClient()
	mux.HandleFunc("GET /api/stream", sc.StreamHandler)

	// API routes
	mux.HandleFunc("GET /api/search", route.NewNavSearchHandler(db))
	mux.HandleFunc("GET /api/anime-timetable", route.NewAnimeTimetableHandler(db))

	frontendFS, err := fs.Sub(embeddedFrontend, "frontend")
	if err != nil {
		werr := lumo.WrapError(err)
		lumo.Panic("Failed to subtree frontend assets: %v", werr)
	}

	// Serve the frontend assets at the root path "/"
	mux.Handle("GET /", http.FileServer(http.FS(frontendFS)))

	w := webview.New(true)
	w.SetTitle("Jubako")
	w.SetSize(1024, 640, webview.HintNone)

	app := &App{
		Ctx:         ctx,
		CancelCtx:   cancel,
		SwarmClient: sc,
		HttpServer: &http.Server{
			Addr:    "127.0.0.1:" + config.HTTP_PORT,
			Handler: mux,
		},
		DB:      db,
		WebView: w,
	}

	app.HttpServer.SetKeepAlivesEnabled(true)
	return app
}

func (app *App) Run() {
	defer app.WebView.Destroy()

	startedAt := time.Now()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	serverErr := make(chan error, 2)
	app.StartedAt = &startedAt

	lumo.Info("Started HTTP server at 127.0.0.1:%s!", config.HTTP_PORT)
	go func() {
		serverErr <- app.HttpServer.ListenAndServe()
	}()

	// Handle signals and context cancellation in a separate goroutine
	go func() {
		select {
		case <-app.Ctx.Done():
			lumo.Info("Received core app context cancellation -> requested to shut down app handler...")
		case sig := <-stop:
			lumo.Info("Received \"%v\" signal -> requested to shut down app handler...", sig)
			app.WebView.Terminate()
		case err := <-serverErr:
			if err != nil && err != http.ErrServerClosed {
				werr := lumo.WrapError(err)
				lumo.Error("Received error from app's internal http server: %v", werr)
				app.WebView.Terminate()
			}
		}
	}()

	app.WebView.Navigate(fmt.Sprintf("http://127.0.0.1:%s/view/index.html", config.HTTP_PORT))

	lumo.Info("Started WebView window - application should be ready.")
	app.WebView.Run()

	app.Shutdown()
}

func (app *App) Shutdown() {
	lumo.Debug("Stopping all services watching main context...")
	app.CancelCtx()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := app.HttpServer.Shutdown(shutdownCtx); err != nil {
		lumo.Warn("Graceful shutdown failed: %v. Force closing HTTP server...\n", err)
		_ = app.HttpServer.Close()
	}

	lumo.Info("Finished shutdown process. Bye!")
}
