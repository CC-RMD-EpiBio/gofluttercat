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
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/irtcat"
	badger "github.com/dgraph-io/badger/v4"
	"github.com/go-chi/chi/v5"
	"github.com/mederrata/ndvek"
	"github.com/swaggest/usecase/status"
)

// buildRbScores reconstructs BayesianScorers from the session state and
// computes Rao-Blackwellized energies using the imputation model.
func (sh SummaryHandler) buildRbScores(rehydrated *irtcat.SessionState, reg *InstrumentRegistry) map[string]*irtcat.BayesianScore {
	scores := make(map[string]*irtcat.BayesianScore)
	grid := ndvek.Linspace(-10, 10, 400)
	for label, energy := range rehydrated.Energies {
		model, ok := reg.Models[label]
		if !ok {
			continue
		}
		scorer := buildScorer(label, reg, rehydrated, grid)

		// Reconstruct answered items for the imputation model
		for _, sr := range rehydrated.Responses {
			for _, it := range model.GetItems() {
				if it.Name == sr.ItemName {
					scorer.Answered = append(scorer.Answered, &irtcat.Response{
						Value: sr.Value,
						Item:  it,
					})
					break
				}
			}
		}

		rbEnergy := scorer.ScoreRaoBlackwell()
		scores[label] = &irtcat.BayesianScore{
			Energy:   energy,
			Grid:     grid,
			RbEnergy: rbEnergy,
		}
	}
	return scores
}

type SummaryHandler struct {
	db          *badger.DB
	instruments map[string]*InstrumentRegistry
	context     *context.Context
}

type SessionSummary struct {
	SessionId      string                   `json:"session_id"`
	StartTime      time.Time                `json:"start_time"`
	ExpirationTime time.Time                `json:"expiration_time"`
	Responses      []*irtcat.SkinnyResponse `json:"responses"`
}

type ScoreSummary struct {
	Deciles   []float64 `json:"deciles"`
	RbDeciles []float64 `json:"rb_deciles"`
	Density   []float64 `json:"density"`
	RbDensity []float64 `json:"rb_density,omitempty"`
	Grid      []float64 `json:"grid"`
	Mean      float64   `json:"mean"`
	Std       float64   `json:"std"`
	RbMean    float64   `json:"rb_mean"`
	RbStd     float64   `json:"rb_std"`
}
type Summary struct {
	Now     time.Time               `header:"X-Now" json:"-"`
	Scores  map[string]ScoreSummary `json:"scores"`
	Session SessionSummary          `json:"session"`
}

func NewSesssionSummary(s irtcat.SessionState) SessionSummary {
	out := SessionSummary{
		SessionId:      s.SessionId,
		StartTime:      s.Start,
		ExpirationTime: s.Expiration,
		Responses:      s.Responses,
	}
	return out
}

func NewScoreSummary(bs *irtcat.BayesianScore) ScoreSummary {
	out := ScoreSummary{
		// Default fields use marginalized posterior when available
		Mean:    bs.Mean(),
		Std:     bs.Std(),
		Deciles: bs.Deciles(),
		Density: bs.Density(),
		Grid:    bs.Grid,
	}
	if len(bs.RbEnergy) > 0 {
		out.RbMean = bs.RbMean()
		out.RbStd = bs.RbStd()
		out.RbDeciles = bs.RbDeciles()
		out.RbDensity = bs.RbDensity()
	}
	return out
}

func NewSummaryHandler(db *badger.DB, instruments map[string]*InstrumentRegistry, ctx context.Context) *SummaryHandler {
	return &SummaryHandler{
		db:          db,
		instruments: instruments,
		context:     &ctx,
	}
}

// getRegistry looks up the InstrumentRegistry for the given session.
func (sh SummaryHandler) getRegistry(session *irtcat.SessionState) *InstrumentRegistry {
	reg, ok := sh.instruments[session.InstrumentID]
	if !ok {
		reg = sh.instruments["rwa"]
	}
	return reg
}

// GetSummary retrieves and computes the summary for a session.
func GetSummary(sid string, db *badger.DB, ctx *context.Context,
	instruments map[string]*InstrumentRegistry) (*Summary, error) {

	rehydrated, err := irtcat.SessionStateFromId(sid, db, ctx)
	if err != nil {
		return nil, fmt.Errorf("%s not found: %w", sid, err)
	}

	reg, ok := instruments[rehydrated.InstrumentID]
	if !ok {
		reg = instruments["rwa"]
	}

	// Build Rao-Blackwellized scores
	grid := ndvek.Linspace(-10, 10, 400)
	scores := make(map[string]*irtcat.BayesianScore)
	for label, energy := range rehydrated.Energies {
		model, ok := reg.Models[label]
		if !ok {
			continue
		}
		scorer := buildScorer(label, reg, rehydrated, grid)
		for _, sr := range rehydrated.Responses {
			for _, it := range model.GetItems() {
				if it.Name == sr.ItemName {
					scorer.Answered = append(scorer.Answered, &irtcat.Response{
						Value: sr.Value,
						Item:  it,
					})
					break
				}
			}
		}
		rbEnergy := scorer.ScoreRaoBlackwell()
		scores[label] = &irtcat.BayesianScore{
			Energy:   energy,
			Grid:     grid,
			RbEnergy: rbEnergy,
		}
	}

	summary := &Summary{
		Session: NewSesssionSummary(*rehydrated),
		Scores:  make(map[string]ScoreSummary),
	}
	for label, bs := range scores {
		summary.Scores[label] = NewScoreSummary(bs)
	}
	return summary, nil
}

func (sh SummaryHandler) ProvideSummary(writer http.ResponseWriter, request *http.Request) {
	sid := chi.URLParam(request, "sid")
	summary, err := GetSummary(sid, sh.db, sh.context, sh.instruments)
	if err != nil {
		RespondWithError(writer, http.StatusNotFound, sid+" not found")
		return
	}
	respondWithJSON(writer, http.StatusOK, summary)
}

type summaryInput struct {
	Locale string `query:"locale" default:"en-US" pattern:"^[a-z]{2}-[A-Z]{2}$" enum:"ru-RU,en-US"`
	Sid    string `path:"sid" minLength:"12"`
}

func (sh SummaryHandler) ProvideSummaryIO(ctx context.Context, input summaryInput, output *Summary) error {
	output.Now = time.Now()
	sid := input.Sid
	rehydrated, err := irtcat.SessionStateFromId(sid, sh.db, sh.context)
	if err != nil {
		return status.Wrap(errors.New("session not found"), status.InvalidArgument)
	}

	reg := sh.getRegistry(rehydrated)
	scores := sh.buildRbScores(rehydrated, reg)
	output.Session = NewSesssionSummary(*rehydrated)
	output.Scores = make(map[string]ScoreSummary)

	for label, bs := range scores {
		output.Scores[label] = NewScoreSummary(bs)
	}
	return nil
}
