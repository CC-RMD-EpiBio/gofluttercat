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
	"io"
	"log"
	"net/http"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/irtcat"
	math2 "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/math"
	badger "github.com/dgraph-io/badger/v4"
	"github.com/go-chi/chi/v5"
	"github.com/mederrata/ndvek"
	"github.com/viterin/vek"
)

type CatHandlerHelper struct {
	db          *badger.DB
	instruments map[string]*InstrumentRegistry
	Context     *context.Context
}

type ItemServed struct {
	Choices  map[string]irtcat.Choice `json:"responses"`
	Name     string                   `json:"name"`
	Question string                   `json:"question"`
	Version  float32                  `json:"version"`
}

func NewCatHandlerHelper(db *badger.DB, instruments map[string]*InstrumentRegistry, context *context.Context) CatHandlerHelper {
	return CatHandlerHelper{
		db:          db,
		instruments: instruments,
		Context:     context,
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

// getRegistry looks up the InstrumentRegistry for the given session.
func (ch *CatHandlerHelper) getRegistry(session *irtcat.SessionState) *InstrumentRegistry {
	reg, ok := ch.instruments[session.InstrumentID]
	if !ok {
		// Fallback to rwa for legacy sessions without InstrumentID
		reg = ch.instruments["rwa"]
	}
	return reg
}

// NextItem chooses a scale randomly and selects the next item
func (ch *CatHandlerHelper) NextItem(writer http.ResponseWriter, request *http.Request) {
	sid := chi.URLParam(request, "sid")
	rehydrated, err := irtcat.SessionStateFromId(sid, ch.db, ch.Context)
	if err != nil {
		log.Printf("err: %v\n", err)
		RespondWithError(writer, http.StatusNotFound, sid+" not found")
		return
	}

	reg := ch.getRegistry(rehydrated)
	if reg == nil {
		RespondWithError(writer, http.StatusInternalServerError, "instrument not found")
		return
	}

	// Check stopping criterion: if all scales have posterior std below
	// threshold and enough items have been answered, stop the assessment.
	catCfg := rehydrated.Config
	if catCfg.StoppingStd > 0 && len(rehydrated.Responses) >= catCfg.MinimumNumItems {
		grid := ndvek.Linspace(-10, 10, 400)
		allConverged := true
		for _, energy := range rehydrated.Energies {
			bs := irtcat.BayesianScore{
				Energy: energy,
				Grid:   grid,
			}
			if bs.Std() > catCfg.StoppingStd {
				allConverged = false
				break
			}
		}
		if allConverged {
			RespondWithError(writer, http.StatusNoContent, "Assessment converged")
			return
		}
	}

	admissibleScales := make([]string, 0)
	for lab := range rehydrated.Energies {
		admissibleScales = append(admissibleScales, lab)
	}
	done := false
	var item *irtcat.Item
	for !done {
		nScales := len(admissibleScales)
		if nScales == 0 {
			break
		}
		scale := admissibleScales[math2.SampleCategorical(vek.DivNumber(vek.Ones(nScales), float64(nScales)))]
		scorer := irtcat.NewBayesianScorer(
			ndvek.Linspace(-10, 10, 400),
			irtcat.DefaultAbilityPrior,
			*reg.Models[scale],
		)
		scorer.ImputationModel = reg.ImputationModel
		scorer.Answered = make([]*irtcat.Response, 0)
		for _, sr := range rehydrated.Responses {
			var itm *irtcat.Item
		medium:
			for _, model := range reg.Models {
				for _, it := range model.GetItems() {
					if it.Name == sr.ItemName {
						itm = it
						break medium
					}
				}
			}
			scorer.Answered = append(scorer.Answered,
				&irtcat.Response{
					Value: sr.Value,
					Item:  itm,
				},
			)
		}
		scorer.Running.Energy = rehydrated.Energies[scale]

		kselector := irtcat.CrossEntropySelector{Temperature: 1.0}
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
	rehydrated, err := irtcat.SessionStateFromId(sid, ch.db, ch.Context)
	if err != nil {
		log.Printf("err: %v\n", err)
		RespondWithError(writer, http.StatusNotFound, sid+" not found")
		return
	}

	reg := ch.getRegistry(rehydrated)
	if reg == nil {
		RespondWithError(writer, http.StatusInternalServerError, "instrument not found")
		return
	}

	scale := chi.URLParam(request, "scale")
	scorer := irtcat.NewBayesianScorer(
		ndvek.Linspace(-10, 10, 400),
		irtcat.DefaultAbilityPrior,
		*reg.Models[scale],
	)
	scorer.ImputationModel = reg.ImputationModel
	scorer.Running.Energy = rehydrated.Energies[scale]

	kselector := irtcat.CrossEntropySelector{Temperature: 0}
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
	rehydrated, err := irtcat.SessionStateFromId(sid, ch.db, ch.Context)
	if err != nil {
		log.Printf("err: %v\n", err)
		RespondWithError(writer, http.StatusNotFound, sid+" not found")
		return
	}

	reg := ch.getRegistry(rehydrated)
	if reg == nil {
		RespondWithError(writer, http.StatusInternalServerError, "instrument not found")
		return
	}

	body, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, "Error reading request body", http.StatusBadRequest)
		return
	}
	defer request.Body.Close()

	var requestData irtcat.SkinnyResponse
	err = json.Unmarshal(body, &requestData)
	if err != nil {
		http.Error(writer, "Invalid JSON", http.StatusBadRequest)
		return
	}

	for scale, model := range reg.Models {
		itm := irtcat.GetItemByName(requestData.ItemName, model.Items)
		if itm != nil {
			resp := irtcat.Response{
				Value: requestData.Value,
				Item:  itm,
			}
			scorer := irtcat.NewBayesianScorer(
				ndvek.Linspace(-10, 10, 400),
				irtcat.DefaultAbilityPrior,
				model,
			)
			scorer.ImputationModel = reg.ImputationModel

			scorer.Running.Energy = rehydrated.Energies[scale]
			scorer.AddResponses([]irtcat.Response{resp})
			rehydrated.Energies[scale] = scorer.Running.Energy
		}
	}

	rehydrated.Responses = append(rehydrated.Responses, &requestData)
	sbyte, _ := rehydrated.ByteMarshal()

	txn := ch.db.NewTransaction(true)
	defer txn.Discard()

	err = txn.Set([]byte(sid), sbyte)
	if err != nil {
		log.Printf("err: %v\n", err)
		RespondWithError(writer, http.StatusInternalServerError, err.Error())
		return
	}

	if err := txn.Commit(); err != nil {
		log.Printf("err: %v\n", err)
		RespondWithError(writer, http.StatusInternalServerError, err.Error())
		return
	}
	respondWithJSON(writer, http.StatusOK, nil)
}
