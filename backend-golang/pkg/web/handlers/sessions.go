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

	cat "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/cat"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/irt"
	badger "github.com/dgraph-io/badger/v4"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/mederrata/ndvek"
)

type SessionHandler struct {
	db       *badger.DB
	models   map[string]*irt.GradedResponseModel
	context  *context.Context
	filePath *string
}

type Answer struct {
	Name     string
	Response int
}

func (sh *SessionHandler) SessionOK(sid string) bool {
	return true
}

func NewSessionHandler(db *badger.DB, models map[string]*irt.GradedResponseModel, ctx context.Context, filePath *string) SessionHandler {
	return SessionHandler{
		db:       db,
		models:   models,
		context:  &ctx,
		filePath: filePath,
	}
}

func (sh *SessionHandler) NewCatSession(writer http.ResponseWriter, request *http.Request) {
	id := uuid.New()

	// initialize the CAT session
	scorers := make(map[string]*irt.BayesianScorer, 0)
	for label, m := range sh.models {
		scorers[label] = irt.NewBayesianScorer(ndvek.Linspace(-10, 10, 400), irt.DefaultAbilityPrior, *m)
	}

	energies := make(map[string][]float64, 0)
	for label, s := range scorers {
		energies[label] = s.Running.Energy
	}

	sess := &cat.SessionState{
		SessionId:  "catsession:" + id.String(),
		Start:      time.Now(),
		Expiration: time.Now().Local().Add(time.Hour * time.Duration(24)),
		Energies:   energies,
		Responses:  make([]*cat.SkinnyResponse, 0),
	}

	sbyte, _ := sess.ByteMarshal()

	if sh.context == nil {
		ctx := context.Background()
		sh.context = &ctx
	}
	err := sh.db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry([]byte(sess.SessionId), sbyte)
		err := txn.SetEntry(e)
		return err
	})

	if err != nil {
		log.Printf("err: %v\n", err)
		RespondWithError(writer, http.StatusInternalServerError, err.Error())
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
	// serialize session to disk and clear from redis
	sid := chi.URLParam(request, "sid")

	rehydrated, err := cat.SessionStateFromId(sid, sh.db, sh.context)
	if err != nil {
		log.Printf("err: %v\n", err)
	}

	rehydrated.Expiration = time.Now()

	if sh.filePath != nil {
		// serialize to disk
	}
	txn := sh.db.NewTransaction(true)
	defer txn.Discard()
	err = txn.Delete([]byte(sid))
	if err != nil {
		RespondWithError(writer, http.StatusNotFound, err.Error())
		return
	}
	// Commit the transaction and check for error.
	if err := txn.Commit(); err != nil {
		RespondWithError(writer, http.StatusNotFound, err.Error())
		return
	}

	respondWithJSON(writer, http.StatusOK, nil)

}

func (sh *SessionHandler) GetSessions(writer http.ResponseWriter, request *http.Request) {
	// Get all active sessions
	var sessionKeys []string

	err := sh.db.View(func(txn *badger.Txn) error {
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
		RespondWithError(writer, http.StatusNotFound, err.Error())
		return
	}
	respondWithJSON(writer, http.StatusOK, sessionKeys)
}
