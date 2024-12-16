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

package handlers

import (
	"context"
	"log"
	"net/http"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/irt"
	math2 "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/math"
	"github.com/go-chi/chi/v5"
	"github.com/mederrata/ndvek"
	"github.com/redis/go-redis/v9"
	"github.com/viterin/vek"
)

type CatHandlerHelper struct {
	rdb     *redis.Client
	models  map[string]*irt.GradedResponseModel
	Context *context.Context
}

func NewCatHandlerHelper(rdb *redis.Client, models map[string]*irt.GradedResponseModel, context *context.Context) CatHandlerHelper {
	return CatHandlerHelper{
		rdb:     rdb,
		models:  models,
		Context: context,
	}
}

// NextItem chooses a scale randomly and selects the next item
func (ch *CatHandlerHelper) NextItem(writer http.ResponseWriter, request *http.Request) {

	sid := chi.URLParam(request, "sid")
	rehydrated, err := models.SessionStateFromId(sid, *ch.rdb, ch.Context)
	if err != nil {
		log.Printf("err: %v\n", err)
	}
	admissibleScales := make([]string, 0)
	for lab, _ := range rehydrated.Energies {
		admissibleScales = append(admissibleScales, lab)
	}
	done := false
	for !done {
		nScales := len(admissibleScales)
		scale := admissibleScales[math2.SampleCategorical(vek.DivNumber(vek.Ones(nScales), float64(nScales)))]
		scorer := models.NewBayesianScorer(
			ndvek.Linspace(-10, 10, 400),
			models.DefaultAbilityPrior,
			*ch.models[scale],
		)
		scorer.Running.Energy = rehydrated.Energies[scale]

	}
}

func (ch *CatHandlerHelper) NextScaleItem(writer http.ResponseWriter, request *http.Request) {
	sid := chi.URLParam(request, "sid")
	rehydrated, err := models.SessionStateFromId(sid, *ch.rdb, ch.Context)
	if err != nil {
		log.Printf("err: %v\n", err)
	}
	scale := chi.URLParam(request, "scale")

}

func (ch *CatHandlerHelper) RegisterResponse(writer http.ResponseWriter, request *http.Request) {
	sid := chi.URLParam(request, "sid")
	rehydrated, err := models.SessionStateFromId(sid, *ch.rdb, ch.Context)
	if err != nil {
		log.Printf("err: %v\n", err)
	}

}
