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

	conf "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/config"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/imputation"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/irtcat"
	badger "github.com/dgraph-io/badger/v4"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/mederrata/ndvek"
)

// InstrumentRegistry holds the loaded models and metadata for one instrument.
type InstrumentRegistry struct {
	Models          map[string]*irtcat.GradedResponseModel
	ImputationModel imputation.ImputationModel
}

type SessionHandler struct {
	db          *badger.DB
	instruments map[string]*InstrumentRegistry
	context     *context.Context
	filePath    *string
	catCfg      conf.CatConfig
}

type Answer struct {
	Name     string
	Response int
}

func (sh *SessionHandler) SessionOK(sid string) bool {
	return true
}

func NewSessionHandler(db *badger.DB, instruments map[string]*InstrumentRegistry, ctx context.Context, filePath *string, catCfg conf.CatConfig) SessionHandler {
	return SessionHandler{
		db:          db,
		instruments: instruments,
		context:     &ctx,
		filePath:    filePath,
		catCfg:      catCfg,
	}
}

type createSessionRequest struct {
	Instrument string `json:"instrument"`
}

// CreateSession creates a new CAT session for the given instrument and persists it.
// Returns the new SessionState or an error.
func CreateSession(instrumentID string, instruments map[string]*InstrumentRegistry,
	db *badger.DB, ctx *context.Context, catCfg conf.CatConfig) (*irtcat.SessionState, error) {

	reg, ok := instruments[instrumentID]
	if !ok {
		return nil, fmt.Errorf("unknown instrument: %s", instrumentID)
	}

	id := uuid.New()

	scorers := make(map[string]*irtcat.BayesianScorer, 0)
	for label, m := range reg.Models {
		scorer := irtcat.NewBayesianScorer(ndvek.Linspace(-10, 10, 400), irtcat.DefaultAbilityPrior, *m)
		scorer.ImputationModel = reg.ImputationModel
		scorers[label] = scorer
	}

	energies := make(map[string][]float64, 0)
	for label, s := range scorers {
		energies[label] = s.Running.Energy
	}

	sess := &irtcat.SessionState{
		SessionId:    "catsession:" + id.String(),
		InstrumentID: instrumentID,
		Start:        time.Now(),
		Expiration:   time.Now().Local().Add(time.Hour * time.Duration(24)),
		Energies:     energies,
		Responses:    make([]*irtcat.SkinnyResponse, 0),
		Config:       catCfg,
	}

	sbyte, _ := sess.ByteMarshal()

	err := db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry([]byte(sess.SessionId), sbyte)
		err := txn.SetEntry(e)
		return err
	})

	if err != nil {
		return nil, err
	}
	log.Printf("New Session: %v (instrument: %s)\n", sess.SessionId, instrumentID)
	return sess, nil
}

func (sh *SessionHandler) NewCatSession(writer http.ResponseWriter, request *http.Request) {
	// Parse optional instrument from request body
	instrumentID := "rwa" // default
	if request.Body != nil {
		body, err := io.ReadAll(request.Body)
		defer request.Body.Close()
		if err == nil && len(body) > 0 {
			var req createSessionRequest
			if json.Unmarshal(body, &req) == nil && req.Instrument != "" {
				instrumentID = req.Instrument
			}
		}
	}

	sess, err := CreateSession(instrumentID, sh.instruments, sh.db, sh.context, sh.catCfg)
	if err != nil {
		RespondWithError(writer, http.StatusBadRequest, err.Error())
		return
	}

	out := map[string]string{
		"session_id":      sess.SessionId,
		"start_time":      sess.Start.String(),
		"expiration_time": sess.Expiration.String(),
	}
	_ = respondWithJSON(writer, http.StatusOK, out)
}

// DeleteSession deletes a session from the database.
func DeleteSession(sid string, db *badger.DB) error {
	txn := db.NewTransaction(true)
	defer txn.Discard()
	err := txn.Delete([]byte(sid))
	if err != nil {
		return err
	}
	return txn.Commit()
}

// ListSessions returns all active session IDs.
func ListSessions(db *badger.DB) ([]string, error) {
	var sessionKeys []string
	err := db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte("catsession:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := item.Key()
			sessionKeys = append(sessionKeys, string(k))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return sessionKeys, nil
}

func (sh *SessionHandler) DeactivateCatSession(writer http.ResponseWriter, request *http.Request) {
	sid := chi.URLParam(request, "sid")
	err := DeleteSession(sid, sh.db)
	if err != nil {
		RespondWithError(writer, http.StatusNotFound, err.Error())
		return
	}
	respondWithJSON(writer, http.StatusOK, nil)
}

func (sh *SessionHandler) GetSessions(writer http.ResponseWriter, request *http.Request) {
	keys, err := ListSessions(sh.db)
	if err != nil {
		RespondWithError(writer, http.StatusNotFound, err.Error())
		return
	}
	respondWithJSON(writer, http.StatusOK, keys)
}
