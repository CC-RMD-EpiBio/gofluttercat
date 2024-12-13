package web

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	conf "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/config"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/irt"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/rwas"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/internal"
	"github.com/alexedwards/scs/v2"
	"github.com/redis/go-redis/v9"
	"github.com/swaggest/rest/openapi"
	"google.golang.org/grpc"
)

var sessionManager *scs.SessionManager

type App struct {
	router    http.Handler
	rdb       *redis.Client
	config    conf.Config
	Models    map[string]irt.GradedResponseModel
	ApiSchema *openapi.Collector
	Context   context.Context
}

func New(config *conf.Config, ctx context.Context) *App {
	// sessionManager.Lifetime = 48 * time.Hour

	app := &App{
		config: *config,
		rdb: redis.NewClient(&redis.Options{
			Addr: config.Redis.Host + ":" + config.Redis.Port,
		}),
		ApiSchema: &openapi.Collector{},
		Models:    rwas.Load(),
		Context:   ctx,
	}
	app.ApiSchema.Reflector().SpecEns().Info.Title = "gofluttercat"
	app.ApiSchema.Reflector().SpecEns().Info.WithDescription("REST API.")
	app.ApiSchema.Reflector().SpecEns().Info.Version = internal.Version
	app.loadRoutes()

	return app
}

func (a *App) StartGRPC(ctx context.Context) error {
	lis, err := net.Listen("tcp", ":3001")
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer()
	if err := grpcServer.Serve(lis); err != nil {
		return err
	}
	return nil
}

func (a *App) Start(ctx context.Context) error {
	server := &http.Server{

		Addr: ":" + a.config.Server.InternalPort,

		Handler: a.router,
	}

	err := a.rdb.Ping(ctx).Err()
	if err != nil {
		return fmt.Errorf("failed to connect to redis: %w", err)
	}
	log.Printf("Connected to Redis at ")

	defer func() {
		if err := a.rdb.Close(); err != nil {
			fmt.Println("failed to close redis", err)
		}
	}()

	fmt.Println("Starting server at " + server.Addr)

	ch := make(chan error, 1)

	go func() {
		err = server.ListenAndServe()
		if err != nil {
			ch <- fmt.Errorf("failed to start server: %w", err)
		}
		close(ch)
	}()

	select {
	case err = <-ch:
		return err
	case <-ctx.Done():
		timeout, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		return server.Shutdown(timeout)
	}

}
