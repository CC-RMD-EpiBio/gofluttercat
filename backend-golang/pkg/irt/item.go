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

package irt

import (
	"encoding/json"
	"log"
	"os"

	"github.com/mederrata/ndvek"
)

type Item struct {
	Name          string                 `json:"name"`
	Question      string                 `json:"question"`
	Choices       map[string]Choice      `json:"responses"`
	ScaleLoadings map[string]Calibration `json:"scales"`
	Version       float32                `json:"version"`
	ScoredValues  []int                  `json:"scored_vales"`
}

type ItemDb struct {
	Items *[]Item
}

type Choice struct {
	Text  string `json:"text"`
	Value uint   `json:"value"`
}

type Response struct {
	Value int
	Item  *Item
}

type Calibration struct {
	Difficulties   []float64 `json:"difficulties"`
	Discrimination float64   `json:"discrimination"`
}

func LoadItem(path string, responses []int) *Item {
	dat, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	item := &Item{
		ScoredValues: responses,
	}
	if err := json.Unmarshal(dat, &item); err != nil {
		log.Fatal(err)
	}
	return item
}

func LoadItemS(dat []byte, responses []int) *Item {

	item := &Item{
		ScoredValues: responses,
	}
	if err := json.Unmarshal(dat, &item); err != nil {
		log.Fatal(err)
	}
	return item
}

func (itm Item) Prob(ability float64) *ndvek.NdArray {
	nScored := len(itm.ScoredValues)
	probs, err := ndvek.NewNdArray([]int{nScored}, nil)
	if err != nil {
		panic(err)
	}
	return probs
}
