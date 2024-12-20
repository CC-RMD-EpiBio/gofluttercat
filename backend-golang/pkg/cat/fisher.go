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

package cat

import (
	"fmt"
	"math"
	"math/rand/v2"

	irt "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/irt"
	math2 "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/math"

	"github.com/mederrata/ndvek"
)

type FisherSelector struct {
	Temperature float64
}

type BayesianFisherSelector struct {
	Temperature float64
}

func sample(weights map[string]float64) string {
	r := rand.Float64()
	var cumulative float64 = 0
	var lastKey string
	var Z float64
	for _, w := range weights {
		Z += w
	}
	for label, prob := range weights {
		cumulative += prob / Z
		lastKey = label
		if r < cumulative {
			return label
		}
	}
	return lastKey
}

func (fs FisherSelector) Criterion(bs *irt.BayesianScorer) map[string]float64 {
	abilities, err := ndvek.NewNdArray([]int{1}, []float64{bs.Running.Mean()})
	if err != nil {
		panic(err)
	}
	fish := bs.Model.FisherInformation(abilities)

	crit := make(map[string]float64, 0)
	for label, value := range fish {
		crit[label] = value.Data[0]
	}

	return crit
}

func (fs FisherSelector) NextItem(bs *irt.BayesianScorer) *irt.Item {

	crit := fs.Criterion(bs)

	var Z float64 = 0
	T := fs.Temperature
	admissable := AdmissibleItems(bs)

	probs := make(map[string]float64, 0)

	for _, item := range admissable {
		probs[item.Name] = crit[item.Name]
	}

	if T == 0 {
		var selected string
		var maxval float64
		for key, value := range probs {
			if value > maxval {
				selected = key
				maxval = value
			}
		}
		return GetItemByName(selected, bs.Model.GetItems())
	}

	for key, value := range probs {
		probs[key] = math.Exp(value / T)
		Z += probs[key]
	}
	for key, _ := range probs {
		probs[key] /= Z
	}
	selected := sample(probs)
	fmt.Printf("selected: %v\n", selected)
	return GetItemByName(selected, bs.Model.GetItems())
}

func itemIn(itemName string, itemList []*irt.Item) bool {
	for _, itm := range itemList {
		if itm.Name == itemName {
			return true
		}
	}
	return false
}

func hasResponse(itemName string, responses []*irt.Response) bool {
	for _, r := range responses {
		if r.Item.Name == itemName {
			return true
		}
	}
	return false
}

func (fs BayesianFisherSelector) Criterion(bs *irt.BayesianScorer) map[string]float64 {
	abilities, err := ndvek.NewNdArray([]int{len(bs.AbilityGridPts)}, bs.AbilityGridPts)
	if err != nil {
		panic(err)
	}
	fish := bs.Model.FisherInformation(abilities)
	density := bs.Running.Density()
	fishB := make(map[string]float64, 0)

	for key, val := range fish {
		if hasResponse(key, bs.Answered) {
			continue
		}
		fishB[key] = math2.Trapz2(density, val.Data)
	}
	return fishB
}

func (fs BayesianFisherSelector) NextItem(bs *irt.BayesianScorer) *irt.Item {
	fishB := fs.Criterion(bs)
	var Z float64 = 0
	T := fs.Temperature
	if T == 0 {
		var selected string
		var maxval float64
		for key, value := range fishB {
			if value > maxval {
				selected = key
				maxval = value
			}
		}
		return GetItemByName(selected, bs.Model.GetItems())
	}
	probs := make(map[string]float64, 0)
	for key, value := range fishB {
		probs[key] = math.Exp(value/T) / Z
		Z += probs[key]
	}
	for key, _ := range probs {
		probs[key] /= Z
	}
	selected := sample(probs)
	fmt.Printf("selected: %v\n", selected)
	return GetItemByName(selected, bs.Model.GetItems())
}
