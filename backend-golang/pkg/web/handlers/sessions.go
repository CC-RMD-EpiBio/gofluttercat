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
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/irt"
	"github.com/google/uuid"
	"github.com/mederrata/ndvek"
	"github.com/redis/go-redis/v9"
)

type SessionHandler struct {
	rdb     *redis.Client
	models  map[string]*irt.GradedResponseModel
	context *context.Context
}

type Answer struct {
	Name     string
	Response int
}

func (sh *SessionHandler) SessionOK(sid string) bool {
	return true
}

func NewSessionHandler(rdb *redis.Client, models map[string]*irt.GradedResponseModel, ctx context.Context) SessionHandler {
	return SessionHandler{
		rdb:     rdb,
		models:  models,
		context: &ctx,
	}
}

func (sh *SessionHandler) NewCatSession(writer http.ResponseWriter, request *http.Request) {
	id := uuid.New()

	// initialize the CAT session
	scorers := make(map[string]*models.BayesianScorer, 0)
	for label, m := range sh.models {
		scorers[label] = models.NewBayesianScorer(ndvek.Linspace(-10, 10, 400), models.DefaultAbilityPrior, *m)
	}

	energies := make(map[string][]float64, 0)
	for label, s := range scorers {
		energies[label] = s.Running.Energy
	}

	sess := &models.SessionState{
		SessionId:  id.String(),
		Start:      time.Now(),
		Expiration: time.Now().Local().Add(time.Hour * time.Duration(24)),
		Energies:   energies,
	}

	sbyte, _ := sess.ByteMarshal()

	if sh.context == nil {
		ctx := context.Background()
		sh.context = &ctx
	}
	stus := sh.rdb.Set(*sh.context, sess.SessionId, sbyte, sess.Expiration.Sub(time.Now()))
	err := stus.Err()
	if err != nil {
		log.Printf("err: %v\n", err)
		panic(err)
	}
	log.Printf("New Session: %v\n", sess.SessionId)

	// write record of this session
	out := map[string]string{
		"session_id":      sess.SessionId,
		"start_time":      sess.Start.String(),
		"expiration_time": sess.Expiration.String(),
	}
	err = respondWithJSON(writer, http.StatusOK, out)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

}

func (sh *SessionHandler) DeactivateCatSession(writer http.ResponseWriter, request *http.Request) {

}