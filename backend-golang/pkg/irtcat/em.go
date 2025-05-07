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

package irtcat

import (
	"fmt"
	"math"
	"slices"

	ndvek "github.com/mederrata/ndvek"
	"github.com/viterin/vek"
)

type BayesianEmScorer struct {
	AbilityGridPts []float64
	Prior          func(float64) float64
	Model          IrtModel
	Answered       []*Response
	Scored         map[string]int
	Running        *BayesianScore
	Exclusions     []string
	Iterations     int
}

func NewBayesianEmScorer(AbilityGridPts []float64, abilityPrior func(float64) float64, model IrtModel) *BayesianEmScorer {

	// initialize the prior
	energy := make([]float64, 0)
	for _, x := range AbilityGridPts {
		density := abilityPrior(x)
		energy = append(energy, math.Log(density))
	}

	bs := &BayesianEmScorer{
		AbilityGridPts: AbilityGridPts,
		Prior:          abilityPrior,
		Model:          model,
		Running: &BayesianScore{
			Grid:   AbilityGridPts,
			Energy: energy,
		},
	}

	return bs
}

func (bs BayesianEmScorer) Score(resp *Responses) error {

	// take care of observed portion
	toAdd := make([]Response, 0)
	toDelete := make([]string, 0)
	for _, r := range resp.Responses {
		past, ok := bs.Scored[r.Item.Name]
		if ok {
			if past == r.Value {
				continue
			} else {
				toDelete = append(toDelete, r.Item.Name)
			}
		}
		toAdd = append(toAdd, r)
	}
	err := bs.RemoveResponses(toDelete)
	if err != nil {
		panic(err)
	}
	err = bs.AddResponses(toAdd)
	if err != nil {
		panic(err)
	}

	// EM iterations

	return nil
}

func (bs *BayesianEmScorer) AddResponses(resp []Response) error {
	if len(resp) == 0 {
		return nil
	}
	abilities, err := ndvek.NewNdArray([]int{len(bs.AbilityGridPts)}, bs.AbilityGridPts)
	if err != nil {
		panic(err)
	}

	ll := bs.Model.LogLikelihood(abilities, resp)
	for _, r := range resp {
		bs.Answered = append(bs.Answered, &r)
	}
	bs.Running.Energy = vek.Add(bs.Running.Energy, ll.Data)

	return nil
}

func (bs *BayesianEmScorer) RemoveResponses(itmNames []string) error {
	toDelete := make([]Response, 0)
	toDeleteNames := make([]string, 0)
	for _, r := range bs.Answered {
		if slices.Contains(itmNames, r.Item.Name) {
			toDelete = append(toDelete, *r)
			toDeleteNames = append(toDeleteNames, r.Item.Name)
		}
	}
	fmt.Printf("toDelete: %v\n", toDelete)
	n := 0
	for _, r := range bs.Answered {
		if !slices.Contains(toDeleteNames, r.Item.Name) {
			bs.Answered[n] = r
			n++
		}
	}
	bs.Answered = bs.Answered[:n]

	abilities, err := ndvek.NewNdArray([]int{len(bs.AbilityGridPts)}, bs.AbilityGridPts)
	if err != nil {
		panic(err)
	}
	ll := bs.Model.LogLikelihood(abilities, toDelete)
	fmt.Printf("ll: %v\n", ll)
	bs.Running.Energy = vek.Sub(bs.Running.Energy, ll.Data)

	return nil
}
