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
	"io"
	"log"
	"net/http"
	"time"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/web/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/swaggest/rest"
	"github.com/swaggest/swgui/v3cdn"

	static "github.com/CC-RMD-EpiBio/gofluttercat/static"
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
	// s := web.NewService(openapi31.NewReflector())
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

	sh := handlers.NewSessionHandler(app.db, app.Models, app.Context, nil)
	router.Post("/session", sh.NewCatSession)
	router.Get("/session", sh.GetSessions)
	router.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/openapi.json", http.StatusSeeOther)
	})
	router.Delete("/{sid}", sh.DeactivateCatSession)
	sumh := handlers.NewSummaryHandler(app.db, app.Models, app.Context)

	// summaryUsecase := usecase.NewInteractor(sumh.ProvideSummaryIO)

	router.Get("/{sid}", sumh.ProvideSummary)

	cath := handlers.NewCatHandlerHelper(app.db, app.Models, &app.Context)
	router.Get("/{sid}/item", cath.NextItem)
	router.Get("/{sid}/{scale}/item", cath.NextScaleItem)
	router.Post("/{sid}/response", cath.RegisterResponse)

	// Swagger UI endpoint at /docs.
	router.Method(http.MethodGet, "/docs/openapi.json", app.ApiSchema)
	router.Mount("/docs", v3cdn.NewHandler(app.ApiSchema.Reflector().Spec.Info.Title,
		"/docs/openapi.json", "/docs"))

	favicon := static.Favicon
	router.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		f, err := favicon.Open("favicon.png")

		if err != nil {
			http.Error(w, "Favicon not found", http.StatusNotFound)
			log.Println("Error opening favicon:", err) // Log the error
			return
		}
		defer f.Close()

		// Set the correct Content-Type. Important for browsers to recognize it.
		w.Header().Set("Content-Type", "image/x-icon")

		// Copy the favicon content to the response.
		_, err = io.Copy(w, f)
		if err != nil {
			http.Error(w, "Error serving favicon", http.StatusInternalServerError)
			log.Println("Error serving favicon:", err) // Log the error
			return
		}
	})
	app.router = router
}
