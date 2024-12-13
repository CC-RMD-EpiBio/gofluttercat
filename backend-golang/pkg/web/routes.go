package web

import (
	"net/http"
	"time"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/web/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/swaggest/rest"
	"github.com/swaggest/swgui/v3cdn"

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

	sh := handlers.NewSessionHandler(app.rdb, app.Models, app.Context)
	router.Post("/", sh.NewCatSession)
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/openapi.json", http.StatusSeeOther)
	})

	sumh := handlers.NewSummaryHandler(app.rdb, app.Models, app.Context)
	router.Get("/{sid}", sumh.ProvideSummary)

	cath := handlers.NewCatHandlerHelper(app.rdb, &app.Models, &app.Context)
	router.Get("/{sid}/item", cath.NextItem)
	router.Get("/{sid}/{scale}/item", cath.NextScaleItem)
	router.Post("/{sid}/response", cath.RegisterResponse)

	// Swagger UI endpoint at /docs.
	router.Method(http.MethodGet, "/docs/openapi.json", app.ApiSchema)
	router.Mount("/docs", v3cdn.NewHandler(app.ApiSchema.Reflector().Spec.Info.Title,
		"/docs/openapi.json", "/docs"))
	app.router = router
}
