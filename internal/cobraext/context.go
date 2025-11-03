package cobraext

import (
	"context"

	"github.com/avivsinai/sabx/internal/config"
	"github.com/avivsinai/sabx/internal/output"
	"github.com/avivsinai/sabx/internal/sabapi"
)

type contextKey string

const appContextKey contextKey = "sabx-app"

// App holds process-wide state bound to Cobra commands.
type App struct {
	Config      *config.Config
	ProfileName string
	Printer     *output.Printer
	Client      *sabapi.Client
	BaseURL     string
}

// WithApp attaches application state to a context.Context.
func WithApp(ctx context.Context, app *App) context.Context {
	return context.WithValue(ctx, appContextKey, app)
}

// From extracts the App from context.
func From(ctx context.Context) (*App, bool) {
	val := ctx.Value(appContextKey)
	if val == nil {
		return nil, false
	}
	app, ok := val.(*App)
	return app, ok
}
