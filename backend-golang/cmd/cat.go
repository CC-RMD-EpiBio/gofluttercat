package cmd

import (
	"context"
	"os"
	"os/signal"

	web "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/web"
)

func launchCat() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	app := web.New()
	err := app.Start(ctx)
	return err
}
