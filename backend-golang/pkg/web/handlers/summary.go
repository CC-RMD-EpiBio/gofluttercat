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
	"net/http"
	"time"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/irtcat"
	badger "github.com/dgraph-io/badger/v4"
	"github.com/go-chi/chi/v5"
	"github.com/mederrata/ndvek"
	"github.com/swaggest/usecase/status"
)

type SummaryHandler struct {
	db      *badger.DB
	models  map[string]*irtcat.GradedResponseModel
	context *context.Context
}

type SessionSummary struct {
	SessionId      string                   `json:"session_id"`
	StartTime      time.Time                `json:"start_time"`
	ExpirationTime time.Time                `json:"expiration_time"`
	Responses      []*irtcat.SkinnyResponse `json:"responses"`
}

type ScoreSummary struct {
	Mean      float64   `json:"mean"`
	Std       float64   `json:"std"`
	Deciles   []float64 `json:"deciles"`
	EmMean    float64   `json:"em_mean"`
	EmStd     float64   `json:"em_std"`
	EmDeciles []float64 `json:"em_deciles"`
	// Density []float64 `json:"density"`
	// Grid    []float64 `json:"grid"`
}
type Summary struct {
	Now     time.Time               `header:"X-Now" json:"-"`
	Session SessionSummary          `json:"session"`
	Scores  map[string]ScoreSummary `json:"scores"`
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
		Mean:   bs.Mean(),
		Std:    bs.Std(),
		EmMean: bs.EmMean(),
		EmStd:  bs.EmStd(),
		// Density: bs.Density(),
		// Grid:    bs.Grid,
		Deciles:   bs.Deciles(),
		EmDeciles: bs.EmDeciles(),
	}
	return out
}

func NewSummaryHandler(db *badger.DB, models map[string]*irtcat.GradedResponseModel, ctx context.Context) *SummaryHandler {
	return &SummaryHandler{
		db:      db,
		models:  models,
		context: &ctx,
	}
}

func (sh SummaryHandler) ProvideSummary(writer http.ResponseWriter, request *http.Request) {
	sid := chi.URLParam(request, "sid")
	rehydrated, err := irtcat.SessionStateFromId(sid, sh.db, sh.context)
	if err != nil {
		RespondWithError(writer, http.StatusNotFound, sid+" not found")
		return
	}
	scores := make(map[string]*irtcat.BayesianScore, 0)
	summary := Summary{
		Session: NewSesssionSummary(*rehydrated),
		Scores:  make(map[string]ScoreSummary),
	}

	for label, energy := range rehydrated.Energies {
		scores[label] = &irtcat.BayesianScore{
			Energy: energy,
			Grid:   ndvek.Linspace(-10, 10, 400),
		}
		summary.Scores[label] = NewScoreSummary(scores[label])
	}

	respondWithJSON(writer, http.StatusOK, summary)

}

type summaryInput struct {
	Locale string `query:"locale" default:"en-US" pattern:"^[a-z]{2}-[A-Z]{2}$" enum:"ru-RU,en-US"`
	Sid    string `path:"sid" minLength:"12"` // Field tags define parameter location and JSON schema constraints.
}

func (sh SummaryHandler) ProvideSummaryIO(ctx context.Context, input summaryInput, output *Summary) error {

	output.Now = time.Now()
	sid := input.Sid
	rehydrated, err := irtcat.SessionStateFromId(sid, sh.db, sh.context)
	if err != nil {
		return status.Wrap(errors.New("session not found"), status.InvalidArgument)
	}

	scores := make(map[string]*irtcat.BayesianScore, 0)
	output.Session = NewSesssionSummary(*rehydrated)
	output.Scores = make(map[string]ScoreSummary)

	for label, energy := range rehydrated.Energies {
		scores[label] = &irtcat.BayesianScore{
			Energy: energy,
			Grid:   ndvek.Linspace(-10, 10, 400),
		}
		output.Scores[label] = NewScoreSummary(scores[label])
	}
	return nil

}
