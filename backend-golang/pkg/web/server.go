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
	"os"
	"time"

	conf "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/config"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/imputation"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/irtcat"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/rwas"
	"github.com/dgraph-io/badger/v4"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/internal"
	"github.com/swaggest/rest/openapi"
)

type AssessmentMeta struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Scales      map[string]string `json:"scales"`
	CatConfig   CatMeta           `json:"cat_config"`
}

type CatMeta struct {
	StoppingStd      float64 `json:"stopping_std"`
	StoppingNumItems int     `json:"stopping_num_items"`
	MinimumNumItems  int     `json:"minimum_num_items"`
}

type App struct {
	router          http.Handler
	Context         context.Context
	db              *badger.DB
	Models          map[string]*irtcat.GradedResponseModel
	ImputationModel *imputation.MiceBayesianLoo
	Assessment      AssessmentMeta
	ApiSchema       *openapi.Collector
	config          conf.Config
}

func New(config *conf.Config, ctx context.Context) *App {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		log.Println(err)
	}

	models := loadModels(config)

	// Load imputation model for Rao-Blackwellization (embedded RWA battery)
	var imputationModel *imputation.MiceBayesianLoo
	if config.Assessment.Source == "" || config.Assessment.Source == "embedded" {
		im, err := rwas.LoadImputationModel()
		if err != nil {
			log.Printf("Warning: failed to load imputation model: %v", err)
		} else {
			log.Println("Loaded embedded imputation model for RWA battery")
			imputationModel = im
		}
	}

	// Build scale display name map, keyed by sc.Name (preserves case from YAML
	// values, since Viper lowercases the map keys but not the string values)
	scaleNames := make(map[string]string)
	for _, sc := range config.Assessment.Scales {
		name := sc.Name
		displayName := sc.DisplayName
		if displayName == "" {
			displayName = name
		}
		scaleNames[name] = displayName
	}

	app := &App{
		config:          *config,
		ApiSchema:       &openapi.Collector{},
		db:              db,
		Models:          models,
		ImputationModel: imputationModel,
		Assessment: AssessmentMeta{
			Name:        config.Assessment.Name,
			Description: config.Assessment.Description,
			Scales:      scaleNames,
			CatConfig: CatMeta{
				StoppingStd:      config.Cat.StoppingStd,
				StoppingNumItems: config.Cat.StoppingNumItems,
				MinimumNumItems:  config.Cat.MinimumNumItems,
			},
		},
		Context: ctx,
	}
	app.ApiSchema.Reflector().SpecEns().Info.Title = "gofluttercat"
	app.ApiSchema.Reflector().SpecEns().Info.WithDescription("REST API.")
	app.ApiSchema.Reflector().SpecEns().Info.Version = internal.Version
	app.loadRoutes()

	return app
}

func loadModels(config *conf.Config) map[string]*irtcat.GradedResponseModel {
	source := config.Assessment.Source
	variant := config.Assessment.Variant

	if source == "" || source == "embedded" {
		if variant == "autoencoded" {
			return loadFromEmbedded(rwas.LoadAutoencodedItems(), config)
		}
		return loadFromEmbedded(rwas.LoadItems(), config)
	}

	if source == "directory" {
		return loadFromDirectory(config)
	}

	log.Printf("Unknown assessment source %q, falling back to embedded", source)
	return loadFromEmbedded(rwas.LoadItems(), config)
}

func loadFromEmbedded(items []*irtcat.Item, config *conf.Config) map[string]*irtcat.GradedResponseModel {
	scales := make(map[string]*irtcat.Scale)
	// Use sc.Name as the map key (preserves case). Viper lowercases the outer
	// YAML map keys, but the items have uppercase scale names like "A", "B".
	for _, sc := range config.Assessment.Scales {
		name := sc.Name
		scales[name] = &irtcat.Scale{
			Name:    name,
			Loc:     sc.Loc,
			Scale:   sc.Scale,
			Version: 1.0,
		}
	}
	// If no scales configured, discover from item calibrations
	if len(scales) == 0 {
		for _, itm := range items {
			for scaleName := range itm.ScaleLoadings {
				if _, ok := scales[scaleName]; !ok {
					scales[scaleName] = &irtcat.Scale{
						Name:  scaleName,
						Loc:   0,
						Scale: 1,
					}
				}
			}
		}
	}

	models := make(map[string]*irtcat.GradedResponseModel)
	for scaleName, scale := range scales {
		scaleItems := make([]*irtcat.Item, 0)
		for _, itm := range items {
			if _, ok := itm.ScaleLoadings[scaleName]; ok {
				scaleItems = append(scaleItems, itm)
			}
		}
		mod := irtcat.NewGRM(scaleItems, *scale)
		models[scaleName] = &mod
	}
	return models
}

func loadFromDirectory(config *conf.Config) map[string]*irtcat.GradedResponseModel {
	itemsDir := config.Assessment.ItemsDir
	if itemsDir == "" {
		log.Fatal("assessment.itemsDir is required when source is 'directory'")
	}

	entries, err := os.ReadDir(itemsDir)
	if err != nil {
		log.Fatalf("Failed to read items directory %s: %v", itemsDir, err)
	}

	var items []*irtcat.Item
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		item := irtcat.LoadItem(itemsDir+"/"+entry.Name(), []int{1, 2, 3, 4, 5, 6, 7, 8, 9})
		if item != nil {
			items = append(items, item)
		}
	}

	var scales map[string]*irtcat.Scale
	if config.Assessment.ScalesFile != "" {
		scales = rwas.LoadScales(config.Assessment.ScalesFile)
	}

	// Merge config-defined scales
	if scales == nil {
		scales = make(map[string]*irtcat.Scale)
	}
	for _, sc := range config.Assessment.Scales {
		name := sc.Name
		if name == "" {
			continue
		}
		scales[name] = &irtcat.Scale{
			Name:    name,
			Loc:     sc.Loc,
			Scale:   sc.Scale,
			Version: 1.0,
		}
	}

	// Auto-discover scales if none configured
	if len(scales) == 0 {
		for _, itm := range items {
			for scaleName := range itm.ScaleLoadings {
				if _, ok := scales[scaleName]; !ok {
					scales[scaleName] = &irtcat.Scale{
						Name:  scaleName,
						Loc:   0,
						Scale: 1,
					}
				}
			}
		}
	}

	models := make(map[string]*irtcat.GradedResponseModel)
	for scaleName, scale := range scales {
		scaleItems := make([]*irtcat.Item, 0)
		for _, itm := range items {
			if _, ok := itm.ScaleLoadings[scaleName]; ok {
				scaleItems = append(scaleItems, itm)
			}
		}
		mod := irtcat.NewGRM(scaleItems, *scale)
		models[scaleName] = &mod
	}
	return models
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
