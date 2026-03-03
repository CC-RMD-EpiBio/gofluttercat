/*
###############################################################################
#
#                           COPYRIGHT NOTICE
#                  Mark O. Hatfield Clinical Research Center
#                       National Institutes of Health
#            United States Department of Health and Human Services
#
# This software was developed and is owned by the National Institutes of
# Health Clinical Center (NIHCC), an agency of the United States Department
# of Health and Human Services, which is making the software available to the
# public for any commercial or non-commercial purpose under the following
# open-source BSD license.
#
# Redistribution and use in source and binary forms, with or without
# modification, are permitted provided that the following conditions are met:
#
# (1) Redistributions of source code must retain this copyright
# notice, this list of conditions and the following disclaimer.
#
# (2) Redistributions in binary form must reproduce this copyright
# notice, this list of conditions and the following disclaimer in the
# documentation and/or other materials provided with the distribution.
#
# (3) Neither the names of the National Institutes of Health Clinical
# Center, the National Institutes of Health, the U.S. Department of
# Health and Human Services, nor the names of any of the software
# developers may be used to endorse or promote products derived from
# this software without specific prior written permission.
#
# (4) Please acknowledge NIHCC as the source of this software by including
# the phrase "Courtesy of the U.S. National Institutes of Health Clinical
# Center"or "Source: U.S. National Institutes of Health Clinical Center."
#
# THIS SOFTWARE IS PROVIDED BY THE U.S. GOVERNMENT AND CONTRIBUTORS "AS
# IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED
# TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A
# PARTICULAR PURPOSE ARE DISCLAIMED.
#
# You are under no obligation whatsoever to provide any bug fixes,
# patches, or upgrades to the features, functionality or performance of
# the source code ("Enhancements") to anyone; however, if you choose to
# make your Enhancements available either publicly, or directly to
# the National Institutes of Health Clinical Center, without imposing a
# separate written license agreement for such Enhancements, then you hereby
# grant the following license: a non-exclusive, royalty-free perpetual license
# to install, use, modify, prepare derivative works, incorporate into
# other computer software, distribute, and sublicense such Enhancements or
# derivative works thereof, in binary and source code form.
#
###############################################################################
*/

package web

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/frontend"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/irtcat"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/web/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/swgui/v3cdn"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"

	static "github.com/CC-RMD-EpiBio/gofluttercat/static"
	"github.com/swaggest/rest/chirouter"
	"github.com/swaggest/rest/jsonschema"
	"github.com/swaggest/rest/request"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/rest/response/gzip"
)

// --- API input/output types for OpenAPI documentation ---

type InstrumentInfo struct {
	Scales      map[string]string `json:"scales"`
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
}

type instrumentsOutput struct {
	Instruments []InstrumentInfo `json:"instruments"`
}

type assessmentInput struct {
	Instrument string `query:"instrument" default:"rwa" description:"Instrument ID"`
}

type createSessionInput struct {
	Instrument string `json:"instrument" default:"rwa" description:"Instrument ID"`
}

type createSessionOutput struct {
	SessionID      string `json:"session_id"`
	StartTime      string `json:"start_time"`
	ExpirationTime string `json:"expiration_time"`
}

type sessionsOutput struct {
	Sessions []string `json:"sessions"`
}

type sidInput struct {
	SID string `path:"sid" minLength:"12" description:"Session ID"`
}

type sidScaleInput struct {
	SID   string `path:"sid" minLength:"12" description:"Session ID"`
	Scale string `path:"scale" description:"Scale name"`
}

type responseInput struct {
	SID      string `path:"sid" minLength:"12" description:"Session ID"`
	ItemName string `json:"item_name" description:"Item name"`
	Value    int    `json:"value" description:"Response value"`
}

type nextItemOutput struct {
	handlers.ItemServed
	usecase.OutputWithNoContent
}

type deleteOutput struct {
	usecase.OutputWithNoContent
}

func (app *App) loadRoutes() {
	validatorFactory := jsonschema.NewFactory(app.ApiSchema, app.ApiSchema)
	decoderFactory := request.NewDecoderFactory()
	decoderFactory.ApplyDefaults = true
	decoderFactory.SetDecoderFunc(rest.ParamInPath, chirouter.PathToURLValues)
	router := chirouter.NewWrapper(chi.NewRouter())
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	router.Use(
		middleware.Recoverer,
		nethttp.OpenAPIMiddleware(app.ApiSchema),
		request.DecoderMiddleware(decoderFactory),
		request.ValidatorMiddleware(validatorFactory),
		response.EncoderMiddleware,
		gzip.Middleware,
	)
	router.Use(middleware.Logger)
	router.Use(middleware.Timeout(60 * time.Second))
	router.Use(render.SetContentType(render.ContentTypeJSON))

	// Build handler-level instrument registries
	handlerInstruments := make(map[string]*handlers.InstrumentRegistry)
	for id, reg := range app.Instruments {
		handlerInstruments[id] = &handlers.InstrumentRegistry{
			Models:          reg.Models,
			ImputationModel: reg.ImputationModel,
		}
	}

	// --- Frontend routes (HTML) ---
	// Plain handlers; they set their own Content-Type: text/html.
	metas := make(map[string]frontend.AssessmentMetaView)
	for id, reg := range app.Instruments {
		metas[id] = frontend.AssessmentMetaView{
			Name:        reg.Meta.Name,
			Description: reg.Meta.Description,
			Scales:      reg.Meta.Scales,
		}
	}
	fh := frontend.NewFrontendHandler(app.db, handlerInstruments, app.Context, metas, app.config.Cat)
	router.Get("/", fh.HandleHome)
	router.Post("/ui/start", fh.HandleStartAssessment)
	router.Get("/ui/assess", fh.HandleAssessmentPage)
	router.Post("/ui/respond", fh.HandleSubmitResponse)
	router.Get("/ui/results", fh.HandleResultsPage)

	// --- API routes (usecase pattern → automatic OpenAPI docs) ---

	// GET /instruments
	listInstruments := usecase.NewInteractor(func(_ context.Context, _ struct{}, output *instrumentsOutput) error {
		for id, reg := range app.Instruments {
			output.Instruments = append(output.Instruments, InstrumentInfo{
				ID:          id,
				Name:        reg.Meta.Name,
				Description: reg.Meta.Description,
				Scales:      reg.Meta.Scales,
			})
		}
		return nil
	})
	listInstruments.SetTitle("List Instruments")
	listInstruments.SetDescription("Returns available CAT instruments and their scales.")
	listInstruments.SetTags("Instruments")
	router.Method(http.MethodGet, "/instruments", nethttp.NewHandler(listInstruments))

	// GET /assessment?instrument=rwa
	getAssessment := usecase.NewInteractor(func(_ context.Context, input assessmentInput, output *AssessmentMeta) error {
		reg, ok := app.Instruments[input.Instrument]
		if !ok {
			return status.Wrap(errors.New("instrument not found"), status.NotFound)
		}
		*output = reg.Meta
		return nil
	})
	getAssessment.SetTitle("Get Assessment Metadata")
	getAssessment.SetDescription("Returns metadata for a specific instrument including scales and CAT configuration.")
	getAssessment.SetTags("Instruments")
	router.Method(http.MethodGet, "/assessment", nethttp.NewHandler(getAssessment))

	// POST /session
	createSession := usecase.NewInteractor(func(_ context.Context, input createSessionInput, output *createSessionOutput) error {
		ctx := app.Context
		sess, err := handlers.CreateSession(input.Instrument, handlerInstruments, app.db, &ctx, app.config.Cat)
		if err != nil {
			return status.Wrap(err, status.InvalidArgument)
		}
		output.SessionID = sess.SessionId
		output.StartTime = sess.Start.String()
		output.ExpirationTime = sess.Expiration.String()
		return nil
	})
	createSession.SetTitle("Create Session")
	createSession.SetDescription("Creates a new CAT session for the specified instrument.")
	createSession.SetTags("Sessions")
	router.Method(http.MethodPost, "/session", nethttp.NewHandler(createSession))

	// GET /session
	listSessions := usecase.NewInteractor(func(_ context.Context, _ struct{}, output *sessionsOutput) error {
		keys, err := handlers.ListSessions(app.db)
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		output.Sessions = keys
		return nil
	})
	listSessions.SetTitle("List Sessions")
	listSessions.SetDescription("Returns all active session IDs.")
	listSessions.SetTags("Sessions")
	router.Method(http.MethodGet, "/session", nethttp.NewHandler(listSessions))

	// DELETE /{sid}
	deleteSession := usecase.NewInteractor(func(_ context.Context, input sidInput, output *deleteOutput) error {
		err := handlers.DeleteSession(input.SID, app.db)
		if err != nil {
			return status.Wrap(err, status.NotFound)
		}
		return nil
	})
	deleteSession.SetTitle("Delete Session")
	deleteSession.SetDescription("Deactivates and removes a CAT session.")
	deleteSession.SetTags("Sessions")
	router.Method(http.MethodDelete, "/{sid}", nethttp.NewHandler(deleteSession))

	// GET /{sid} — summary
	getSummary := usecase.NewInteractor(func(_ context.Context, input sidInput, output *handlers.Summary) error {
		ctx := app.Context
		s, err := handlers.GetSummary(input.SID, app.db, &ctx, handlerInstruments)
		if err != nil {
			return status.Wrap(err, status.NotFound)
		}
		*output = *s
		return nil
	})
	getSummary.SetTitle("Get Session Summary")
	getSummary.SetDescription("Returns assessment scores and Rao-Blackwellized summary for a session.")
	getSummary.SetTags("Assessment")
	router.Method(http.MethodGet, "/{sid}", nethttp.NewHandler(getSummary))

	// GET /{sid}/item
	getNextItem := usecase.NewInteractor(func(_ context.Context, input sidInput, output *nextItemOutput) error {
		ctx := app.Context
		item, err := handlers.GetNextItem(input.SID, app.db, &ctx, handlerInstruments)
		if err != nil {
			return status.Wrap(err, status.NotFound)
		}
		if item == nil {
			return nil // 204 No Content via OutputWithNoContent
		}
		output.SetNoContent(false)
		output.ItemServed = *item
		return nil
	})
	getNextItem.SetTitle("Get Next Item")
	getNextItem.SetDescription("Selects the next adaptive item for the session. Returns 204 when assessment is complete.")
	getNextItem.SetTags("Assessment")
	router.Method(http.MethodGet, "/{sid}/item", nethttp.NewHandler(getNextItem))

	// GET /{sid}/{scale}/item
	getNextScaleItem := usecase.NewInteractor(func(_ context.Context, input sidScaleInput, output *nextItemOutput) error {
		ctx := app.Context
		item, err := handlers.GetNextScaleItem(input.SID, input.Scale, app.db, &ctx, handlerInstruments)
		if err != nil {
			return status.Wrap(err, status.NotFound)
		}
		if item == nil {
			return nil // 204 No Content
		}
		output.SetNoContent(false)
		output.ItemServed = *item
		return nil
	})
	getNextScaleItem.SetTitle("Get Next Scale Item")
	getNextScaleItem.SetDescription("Selects the next item for a specific scale (deterministic). Returns 204 when out of items.")
	getNextScaleItem.SetTags("Assessment")
	router.Method(http.MethodGet, "/{sid}/{scale}/item", nethttp.NewHandler(getNextScaleItem))

	// POST /{sid}/response
	registerResponse := usecase.NewInteractor(func(_ context.Context, input responseInput, output *deleteOutput) error {
		ctx := app.Context
		err := handlers.ProcessResponse(input.SID, irtcat.SkinnyResponse{
			ItemName: input.ItemName,
			Value:    input.Value,
		}, app.db, &ctx, handlerInstruments)
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		return nil
	})
	registerResponse.SetTitle("Register Response")
	registerResponse.SetDescription("Registers an item response and updates the session posterior.")
	registerResponse.SetTags("Assessment")
	router.Method(http.MethodPost, "/{sid}/response", nethttp.NewHandler(registerResponse))

	// Swagger UI
	router.Method(http.MethodGet, "/docs/openapi.json", app.ApiSchema)
	router.Mount("/docs", v3cdn.NewHandler(app.ApiSchema.Reflector().Spec.Info.Title,
		"/docs/openapi.json", "/docs"))

	favicon := static.Favicon
	router.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		f, err := favicon.Open("favicon.png")
		if err != nil {
			http.Error(w, "Favicon not found", http.StatusNotFound)
			log.Println("Error opening favicon:", err)
			return
		}
		defer f.Close()
		w.Header().Set("Content-Type", "image/x-icon")
		_, err = io.Copy(w, f)
		if err != nil {
			http.Error(w, "Error serving favicon", http.StatusInternalServerError)
			log.Println("Error serving favicon:", err)
			return
		}
	})
	app.router = router
}
