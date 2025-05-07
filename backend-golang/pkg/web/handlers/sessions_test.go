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
	"log"
	"testing"
	"time"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/config"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/irtcat"

	"github.com/google/uuid"
	"github.com/mederrata/ndvek"
	"github.com/redis/go-redis/v9"
)

func Test_rdb_sessions(t *testing.T) {
	data := ndvek.Linspace(-10, 10, 400)
	energies := make(map[string][]float64, 0)
	energies["A"] = data
	sess := &irtcat.SessionState{
		SessionId:  uuid.New().String(),
		Start:      time.Now(),
		Expiration: time.Now().Local().Add(time.Hour * time.Duration(24)),
		Energies:   energies,
	}

	out, _ := json.Marshal(sess)
	fmt.Printf("out: %v\n", string(out))
	bout, _ := sess.ByteMarshal()
	fmt.Printf("bout: %v\n", bout)
	rehyrdated, _ := irtcat.SessionStateByteUnmarshal(bout)
	fmt.Printf("rehyrdated: %v\n", rehyrdated)

	conf := config.GetConfig()
	rdb := redis.NewClient(&redis.Options{
		Addr: conf.Redis.Host + ":" + conf.Redis.Port,
	})

	ctx := context.Background()
	err := rdb.Ping(ctx).Err()
	if err != nil {
		fmt.Println("failed to connect to redis: %w", err)
	}
	log.Printf("Connected to Redis at ")

	defer func() {
		if err := rdb.Close(); err != nil {
			fmt.Println("failed to close redis", err)
		}
	}()
	stus := rdb.Set(ctx, sess.SessionId, bout, sess.Expiration.Sub(time.Now()))
	fmt.Printf("stus: %v\n", stus)

}
