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

package tests

import (
	"fmt"
	"math"
	"testing"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/cat"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/irt"
	"github.com/mederrata/ndvek"
)

func Test_grm(t *testing.T) {
	item1 := irt.Item{
		Name:     "Item1",
		Question: "I can walk",
		Choices: map[string]irt.Choice{
			"Never": irt.Choice{
				Text: "Never", Value: 1,
			},
			"Sometimes": irt.Choice{
				Text:  "Sometimes",
				Value: 2},
			"Usually":       irt.Choice{Text: "Usually", Value: 3},
			"Almost Always": irt.Choice{Text: "Almost always", Value: 4},
			"Always":        irt.Choice{Text: "Always", Value: 5}},
		ScaleLoadings: map[string]irt.Calibration{
			"default": irt.Calibration{
				Difficulties:   []float64{-2, -1, 1, 2},
				Discrimination: 1.0,
			},
		},
		ScoredValues: []int{1, 2, 3, 4, 5},
	}
	item2 := irt.Item{
		Name:     "Item2",
		Question: "I can run",
		Choices: map[string]irt.Choice{
			"Never": irt.Choice{
				Text: "Never", Value: 1,
			},
			"Sometimes": irt.Choice{
				Text:  "Sometimes",
				Value: 2},
			"Usually":       irt.Choice{Text: "Usually", Value: 3},
			"Almost Always": irt.Choice{Text: "Almost always", Value: 4},
			"Always":        irt.Choice{Text: "Always", Value: 5}},
		ScaleLoadings: map[string]irt.Calibration{
			"default": irt.Calibration{
				Difficulties:   []float64{-1, -0.5, 1.5, 2.5},
				Discrimination: 2.0,
			},
		},
		ScoredValues: []int{1, 2, 3, 4, 5},
	}
	item3 := irt.Item{
		Name:     "Item3",
		Question: "I can jump",
		Choices: map[string]irt.Choice{
			"Never": irt.Choice{
				Text: "Never", Value: 1,
			},
			"Sometimes": irt.Choice{
				Text:  "Sometimes",
				Value: 2},
			"Usually":       irt.Choice{Text: "Usually", Value: 3},
			"Almost Always": irt.Choice{Text: "Almost always", Value: 4},
			"Always":        irt.Choice{Text: "Always", Value: 5}},
		ScaleLoadings: map[string]irt.Calibration{
			"default": irt.Calibration{
				Difficulties:   []float64{0, 1, 2.5, 3.5},
				Discrimination: 3,
			},
		},
		ScoredValues: []int{1, 2, 3, 4, 5},
	}
	scale := irt.Scale{
		Loc:   0,
		Scale: 1,
		Name:  "default",
	}
	grm := irt.NewGRM(
		[]*irt.Item{&item1, &item2, &item3},
		scale,
	)

	resp := irt.Response{
		Value: 1,
		Item:  &item1,
	}

	sresponses := irt.Responses{
		Responses: []irt.Response{resp},
	}

	fmt.Printf("grm: %v\n", grm)

	abilities, err := ndvek.NewNdArray([]int{4}, []float64{0, -1, 1, 2})
	if err != nil {
		panic(err)
	}
	probs := grm.Prob(abilities)
	fmt.Printf("probs: %v\n", probs)
	ll := grm.LogLikelihood(abilities, sresponses.Responses)
	fmt.Printf("ll: %v\n", ll)
	prior := func(x float64) float64 {
		out := math.Exp(-x * x / 2)
		return out
	}

	scorer := irt.NewBayesianScorer(ndvek.Linspace(-10, 10, 400), prior, grm)
	_ = scorer.Score(&sresponses)
	fmt.Printf("scorer.Running: %v\n", scorer.Running.Mean())

	fmt.Printf("\"MCMC selector\": %v\n", "MCMC selector")
	mckselector := cat.NewMcKlSelector(0, 32)
	crit := mckselector.Criterion(scorer)
	fmt.Printf("crit: %v\n", crit)
	mckitem := mckselector.NextItem(scorer)
	fmt.Printf("item: %v\n", mckitem.Name)

	fmt.Printf("\"KL plug-in selector\": %v\n", "KL plug-in selector")
	kselector := cat.KLSelector{Temperature: 0}
	crit = kselector.Criterion(scorer)
	fmt.Printf("crit: %v\n", crit)
	kitem := kselector.NextItem(scorer)
	fmt.Printf("item: %v\n", kitem.Name)

	selector := cat.FisherSelector{Temperature: 0}
	crit = selector.Criterion(scorer)
	fmt.Printf("crit: %v\n", crit)
	item := selector.NextItem(scorer)
	fmt.Printf("item: %v\n", item.Name)

	bselector := cat.BayesianFisherSelector{Temperature: 0}
	crit = bselector.Criterion(scorer)
	fmt.Printf("crit: %v\n", crit)
	bitem := bselector.NextItem(scorer)
	fmt.Printf("item: %v\n", bitem.Name)

}
