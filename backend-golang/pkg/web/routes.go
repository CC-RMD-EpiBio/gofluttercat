package web

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/chirouter"
	"github.com/swaggest/rest/jsonschema"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/request"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/rest/response/gzip"
)

func (app *App) loadRoutes() {
	validatorFactory := jsonschema.NewFactory(app.ApiSchema, app.ApiSchema)
	decoderFactory := request.NewDecoderFactory()
	decoderFactory.ApplyDefaults = true
	decoderFactory.SetDecoderFunc(rest.ParamInPath, chirouter.PathToURLValues)
	router := chirouter.NewWrapper(chi.NewRouter())
	router.Use(
		middleware.Recoverer,                          // Panic recovery.
		nethttp.OpenAPIMiddleware(app.ApiSchema),      // Documentation collector.
		request.DecoderMiddleware(decoderFactory),     // Request decoder setup.
		request.ValidatorMiddleware(validatorFactory), // Request validator setup.
		response.EncoderMiddleware,                    // Response encoder setup.
		gzip.Middleware,                               // Response compression with support for direct gzip pass through.
	)
	router.Use(middleware.Logger)
	router.Use(middleware.Timeout(60 * time.Second))
	router.Use(render.SetContentType(render.ContentTypeJSON))
	router.Post("/", func(w http.ResponseWriter, r *http.Request) {

	})
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		id := uuid.New()
		fmt.Printf("id: %v\n", id)
		w.WriteHeader(http.StatusOK)
	})

	router.Post("/session", func(w http.ResponseWriter, r *http.Request) {
		id := uuid.New()
		fmt.Printf("id: %v\n", id)
		w.WriteHeader(http.StatusOK)
	})
	app.router = router
}
