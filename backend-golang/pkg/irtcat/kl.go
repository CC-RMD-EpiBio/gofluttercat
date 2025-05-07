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
	"math"

	math2 "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/math"
	"github.com/mederrata/ndvek"
	"github.com/viterin/vek"
)

type KLSelector struct {
	Temperature    float64
	SurrogateModel *IrtModel
	Stopping       func() map[string]bool
	EmIters        int
}

func NewKlSelector(temp float64, iters int) KLSelector {
	out := KLSelector{
		Temperature: temp,
		EmIters:     iters,
	}
	return out
}

func (ks KLSelector) Criterion(bs *BayesianScorer) map[string]float64 {
	abilities, err := ndvek.NewNdArray([]int{len(bs.AbilityGridPts)}, bs.AbilityGridPts)
	if err != nil {
		panic(err)
	}
	nAbilities := abilities.Shape()[0]

	probs := bs.Model.Prob(abilities)
	admissable := AdmissibleItems(bs)
	pi_t := bs.Running.Density()
	lpi_t := bs.Running.Energy

	q_z := make(map[string][]float64)

	q_theta := make([]float64, len(pi_t))
	copy(q_theta, pi_t)

	// allocate the arrays
	for _, itm := range admissable {
		pr := probs[itm.Name]
		K := pr.Shape()[1]
		for j := 0; j < nAbilities; j++ {
			q_z[itm.Name] = make([]float64, K)
		}
	}

	// compute log_pi_infty for plugin estimator
	// Now compute Eq (8)
	for _ = range ks.EmIters {
		lpi_z := make([]float64, len(bs.AbilityGridPts))

		for label, _ := range q_z {
			p := probs[label]
			for k := 0; k < p.Shape()[1]; k++ {
				integrand := make([]float64, len(bs.AbilityGridPts))
				for i := 0; i < len(bs.AbilityGridPts); i++ {
					integrand[i], _ = probs[label].Get([]int{i, k})
					integrand[i] *= q_theta[i]
				}
				q_z[label][k] = math2.Trapz2(integrand, bs.AbilityGridPts)

				for i := 0; i < len(bs.AbilityGridPts); i++ {
					pp, _ := p.Get([]int{i, k})
					lpi_z[i] += math2.Xlogy(q_z[label][k], pp)
				}
			}
		}

		lq_theta := vek.Add(lpi_t, lpi_z)
		q_theta = math2.EnergyToDensity(lq_theta, bs.AbilityGridPts)

	}

	deltaItem := make(map[string]float64, 0)

	for _, itm := range admissable {
		// build KL divergence for item
		p := probs[itm.Name]
		deltaItem[itm.Name] = 0
		for k := 0; k < p.Shape()[1]; k++ {
			// make pi_{t+1}
			lpi_next := make([]float64, len(bs.AbilityGridPts))
			copy(lpi_next, lpi_t)
			for i := 0; i < len(bs.AbilityGridPts); i++ {
				pp, _ := p.Get([]int{i, k})
				lpi_next[i] += pp
			}
			pi_next := math2.EnergyToDensity(lpi_next, bs.AbilityGridPts)
			deltaItem[itm.Name] += q_z[itm.Name][k] * math2.KlDivergence(q_theta, pi_next, bs.AbilityGridPts)

		}

	}
	return deltaItem
}

func (ks KLSelector) NextItem(bs *BayesianScorer) *Item {
	deltaItem := ks.Criterion(bs)
	T := ks.Temperature
	admissible := AdmissibleItems(bs)

	probs := make(map[string]float64, 0)

	for _, item := range admissible {
		probs[item.Name] = deltaItem[item.Name]
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

	selectionProbs := make(map[string]float64)
	for key, value := range deltaItem {
		selectionProbs[key] = math.Exp(value / T)
	}

	selected := sample(selectionProbs)
	return GetItemByName(selected, bs.Model.GetItems())
}
