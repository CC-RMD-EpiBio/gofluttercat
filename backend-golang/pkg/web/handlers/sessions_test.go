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
	"testing"
	"time"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/irtcat"
	badger "github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"
	"github.com/mederrata/ndvek"
)

func Test_badger_sessions(t *testing.T) {
	data := ndvek.Linspace(-10, 10, 400)
	energies := make(map[string][]float64, 0)
	energies["A"] = data
	sess := &irtcat.SessionState{
		SessionId:  "catsession:" + uuid.New().String(),
		Start:      time.Now(),
		Expiration: time.Now().Local().Add(time.Hour * time.Duration(24)),
		Energies:   energies,
		Responses:  make([]*irtcat.SkinnyResponse, 0),
	}

	// Test JSON serialization
	out, err := json.Marshal(sess)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("json.Marshal produced empty output")
	}

	// Test gob serialization
	bout, err := sess.ByteMarshal()
	if err != nil {
		t.Fatalf("ByteMarshal failed: %v", err)
	}

	rehydrated, err := irtcat.SessionStateByteUnmarshal(bout)
	if err != nil {
		t.Fatalf("SessionStateByteUnmarshal failed: %v", err)
	}
	if rehydrated.SessionId != sess.SessionId {
		t.Errorf("SessionId mismatch: got %q, want %q", rehydrated.SessionId, sess.SessionId)
	}

	// Test round-trip through Badger
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		t.Fatalf("Failed to open badger: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Write session
	err = db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(sess.SessionId), bout)
	})
	if err != nil {
		t.Fatalf("Failed to write session to badger: %v", err)
	}

	// Read session back
	retrieved, err := irtcat.SessionStateFromId(sess.SessionId, db, &ctx)
	if err != nil {
		t.Fatalf("SessionStateFromId failed: %v", err)
	}
	if retrieved.SessionId != sess.SessionId {
		t.Errorf("Retrieved SessionId mismatch: got %q, want %q", retrieved.SessionId, sess.SessionId)
	}
	if len(retrieved.Energies["A"]) != 400 {
		t.Errorf("Energies length mismatch: got %d, want 400", len(retrieved.Energies["A"]))
	}
}
