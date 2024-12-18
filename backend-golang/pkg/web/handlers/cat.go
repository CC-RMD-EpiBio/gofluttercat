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
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models"
	cat "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/cat"
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

type ItemServed struct {
	Name     string                   `json:"name"`
	Question string                   `json:"question"`
	Choices  map[string]models.Choice `json:"responses"`
	Version  float32                  `json:"version"`
}

func NewCatHandlerHelper(rdb *redis.Client, models map[string]*irt.GradedResponseModel, context *context.Context) CatHandlerHelper {
	return CatHandlerHelper{
		rdb:     rdb,
		models:  models,
		Context: context,
	}
}

func removeStringInPlace(slice []string, strToRemove string) []string {
	var i int
	for _, str := range slice {
		if str != strToRemove {
			slice[i] = str
			i++
		}
	}
	return slice[:i]
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
	var item *models.Item
	for !done {
		nScales := len(admissibleScales)
		scale := admissibleScales[math2.SampleCategorical(vek.DivNumber(vek.Ones(nScales), float64(nScales)))]
		scorer := models.NewBayesianScorer(
			ndvek.Linspace(-10, 10, 400),
			models.DefaultAbilityPrior,
			*ch.models[scale],
		)
		scorer.Answered = rehydrated.Responses
		scorer.Running.Energy = rehydrated.Energies[scale]

		kselector := cat.KLSelector{Temperature: 0}
		item = kselector.NextItem(scorer)
		if item != nil {
			done = true
			break
		}

		admissibleScales = removeStringInPlace(admissibleScales, scale)

	}
	if item == nil {
		RespondWithError(writer, http.StatusNoContent, "Out of items")
		return
	}
	toReturn := ItemServed{
		Name:     item.Name,
		Question: item.Question,
		Choices:  item.Choices,
		Version:  item.Version,
	}

	respondWithJSON(writer, http.StatusOK, toReturn)
}

func (ch *CatHandlerHelper) NextScaleItem(writer http.ResponseWriter, request *http.Request) {
	sid := chi.URLParam(request, "sid")

	rehydrated, err := models.SessionStateFromId(sid, *ch.rdb, ch.Context)
	if err != nil {
		log.Printf("err: %v\n", err)
	}
	scale := chi.URLParam(request, "scale")
	scorer := models.NewBayesianScorer(
		ndvek.Linspace(-10, 10, 400),
		models.DefaultAbilityPrior,
		*ch.models[scale],
	)
	scorer.Running.Energy = rehydrated.Energies[scale]

	kselector := cat.KLSelector{Temperature: 0}
	item := kselector.NextItem(scorer)
	if item == nil {
		RespondWithError(writer, http.StatusNoContent, "Out of items")
		return
	}
	toReturn := ItemServed{
		Name:     item.Name,
		Question: item.Question,
		Choices:  item.Choices,
		Version:  item.Version,
	}

	respondWithJSON(writer, http.StatusOK, toReturn)

}

func (ch *CatHandlerHelper) RegisterResponse(writer http.ResponseWriter, request *http.Request) {
	sid := chi.URLParam(request, "sid")
	rehydrated, err := models.SessionStateFromId(sid, *ch.rdb, ch.Context)
	if err != nil {
		log.Printf("err: %v\n", err)
	}
	body, err := io.ReadAll(request.Body)

	if err != nil {
		http.Error(writer, "Error reading request body", http.StatusBadRequest)
		return
	}
	defer request.Body.Close()

	var requestData models.SkinnyResponse
	err = json.Unmarshal(body, &requestData)
	if err != nil {
		fmt.Printf("body: %v\n", string(body))
		fmt.Printf("err: %v\n", err)
		http.Error(writer, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if rehydrated.Responses == nil {
		rehydrated.Responses = make([]*models.SkinnyResponse, 0)
	}

	//
	for scale, model := range ch.models {
		itm := cat.GetItemByName(requestData.ItemName, model.Items)
		if itm != nil {
			resp := models.Response{
				Value: requestData.Value,
				Item:  itm,
			}
			scorer := models.NewBayesianScorer(
				ndvek.Linspace(-10, 10, 400),
				models.DefaultAbilityPrior,
				model,
			)
			scorer.Running.Energy = rehydrated.Energies[scale]
			scorer.AddResponses([]models.Response{resp})
			rehydrated.Energies[scale] = scorer.Running.Energy
		}
	}

	rehydrated.Responses = append(rehydrated.Responses, &requestData)
	sbyte, _ := rehydrated.ByteMarshal()

	stus := ch.rdb.Set(*ch.Context, sid, sbyte, rehydrated.Expiration.Sub(time.Now()))
	err = stus.Err()
	if err != nil {
		log.Printf("err: %v\n", err)
		RespondWithError(writer, http.StatusInternalServerError, err.Error())
		return
	}
	respondWithJSON(writer, http.StatusOK, nil)

}
