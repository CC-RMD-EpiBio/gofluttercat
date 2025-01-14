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
	"fmt"
	"log"
	"net/http"
	"time"

	conf "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/config"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/irt"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/rwas"
	"github.com/dgraph-io/badger/v4"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/internal"
	"github.com/alexedwards/scs/v2"
	"github.com/swaggest/rest/openapi"
)

var sessionManager *scs.SessionManager

type App struct {
	router    http.Handler
	db        *badger.DB
	config    conf.Config
	Models    map[string]*irt.GradedResponseModel
	ApiSchema *openapi.Collector
	Context   context.Context
}

func New(config *conf.Config, ctx context.Context) *App {
	// sessionManager.Lifetime = 48 * time.Hour
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		log.Println(err)
	}

	app := &App{
		config:    *config,
		ApiSchema: &openapi.Collector{},
		db:        db,
		Models:    rwas.Load(),
		Context:   ctx,
	}
	app.ApiSchema.Reflector().SpecEns().Info.Title = "gofluttercat"
	app.ApiSchema.Reflector().SpecEns().Info.WithDescription("REST API.")
	app.ApiSchema.Reflector().SpecEns().Info.Version = internal.Version
	app.loadRoutes()

	return app
}

func (a *App) Start(ctx context.Context) error {
	server := &http.Server{

		Addr: ":" + a.config.Server.InternalPort,

		Handler: a.router,
	}
	/*
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
	*/

	log.Println("Starting backend server at " + server.Addr)

	ch := make(chan error, 1)
	var err error
	go func() {
		err = server.ListenAndServe()
		if err != nil {
			ch <- fmt.Errorf("failed to start server: %w", err)
		}
		close(ch)
	}()

	select {
	case err = <-ch:
		return nil
	case <-ctx.Done():
		timeout, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		defer a.db.Close()
		return server.Shutdown(timeout)
	}

}
