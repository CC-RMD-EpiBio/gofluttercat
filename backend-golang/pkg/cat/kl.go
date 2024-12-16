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
	"math"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models"
	math2 "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/math"
	"github.com/mederrata/ndvek"
	"github.com/viterin/vek"
)

type KLSelector struct {
	Temperature    float64
	SurrogateModel *models.IrtModel
	Stopping       func() map[string]bool
}

func (ks KLSelector) Criterion(bs *models.BayesianScorer) map[string]float64 {
	abilities, err := ndvek.NewNdArray([]int{len(bs.AbilityGridPts)}, bs.AbilityGridPts)
	if err != nil {
		panic(err)
	}
	nAbilities := abilities.Shape()[0]

	probs := bs.Model.Prob(abilities)
	admissable := make([]*models.Item, 0)
	answered := make([]*models.Item, 0)
	for _, itm := range bs.Model.GetItems() {
		if hasResponse(itm.Name, bs.Answered) {
			answered = append(answered, itm)
			continue
		}
		admissable = append(admissable, itm)
	}
	piAlpha := bs.Running.Density()
	lpInfy := bs.Running.Energy // log pi_{\alpha_t}
	for a, itm := range admissable {
		pr := probs[itm.Name]
		K := pr.Shape()[1]
		for j := 0; j < nAbilities; j++ {
			for k := 0; k < K; k++ {
				val, err := pr.Get([]int{j, k})
				if err != nil {
					panic(err)
				}
				lpInfy[a] += math2.Xlogy(val, val)
			}
		}
	}
	// compute log_pi_infty for plugin estimator
	pi_infty := math2.EnergyToDensity(lpInfy, bs.AbilityGridPts)
	// Now compute Eq (8)
	deltaItem := make(map[string]float64, 0)

	for itm, p := range probs {
		var lpItem float64 = 0
		for k := 0; k < p.Shape()[1]; k++ {
			integrand1 := make([]float64, len(bs.AbilityGridPts))
			integrand2 := make([]float64, len(bs.AbilityGridPts))

			for i := 0; i < len(bs.AbilityGridPts); i++ {
				ell, err := p.Get([]int{i, k})
				if err != nil {
					panic(err)
				}
				if ell < math.SmallestNonzeroFloat64 {
					ell = math.SmallestNonzeroFloat64
				}
				integrand1[i] = math2.Xlogy(pi_infty[i], ell)
				integrand2[i] = ell * piAlpha[i]
			}
			integral1 := math2.Trapz2(integrand1, bs.AbilityGridPts)
			integral2 := math2.Trapz2(integrand2, bs.AbilityGridPts)
			delta := (integral1 - math.Log(integral2))
			lpItem += integral2 * delta
		}
		deltaItem[itm] = -lpItem
	}
	return deltaItem
}

func (ks KLSelector) NextItem(bs *models.BayesianScorer) *models.Item {
	deltaItem := ks.Criterion(bs)
	T := ks.Temperature

	if T == 0 {
		var selected string
		var maxval float64
		for key, value := range deltaItem {
			if value > maxval {
				selected = key
				maxval = value
			}
		}
		return getItemByName(selected, bs.Model.GetItems())
	}

	selectionProbs := make(map[string]float64)
	for key, value := range deltaItem {
		selectionProbs[key] = math.Exp(value / T)
	}

	selected := sample(selectionProbs)
	return getItemByName(selected, bs.Model.GetItems())
}

type McKlSelector struct {
	Temperature float64
	NumSamples  int
}

func NewMcKlSelector(temperature float64, nsamples int) McKlSelector {
	return McKlSelector{
		Temperature: temperature, NumSamples: nsamples}
}

func (ks McKlSelector) Criterion(bs *models.BayesianScorer) map[string]float64 {
	crit := make(map[string]float64, 0)
	abilitySamples := bs.Running.Sample(ks.NumSamples)
	abilitySamplesVek, _ := ndvek.NewNdArray([]int{len(abilitySamples)}, abilitySamples)
	piAlphat := bs.Running.Density()
	abilitiesGrid, _ := ndvek.NewNdArray([]int{len(bs.AbilityGridPts)}, bs.AbilityGridPts)
	samples := bs.Model.Sample(abilitySamplesVek)

	ellTheta := bs.Model.Prob(abilitiesGrid)

	// compute the expected value of ell for each sampled response against $\pi_{\alpha_t}$
	expectedEll := make(map[string][]float64, 0)
	for itm, probs := range ellTheta {
		expectedEll[itm] = make([]float64, 0)
		nChoices := probs.Shape()[1]
		nPts := len(bs.AbilityGridPts)

		for k := range nChoices {
			integrand := make([]float64, nPts)
			for i := range nPts { // i is a grid point
				integrand[i], _ = probs.Get([]int{i, k})
			}
			integrand = vek.Mul(integrand, piAlphat)
			integral := math2.Trapz2(integrand, bs.AbilityGridPts)
			expectedEll[itm] = append(expectedEll[itm], integral)
		}
	}

	for s, _ := range abilitySamples {
		// Computing the integral
		// compute pi_infty
		lpInfty := make([]float64, len(bs.AbilityGridPts))
		for itm, choices := range samples {
			for i := range len(bs.AbilityGridPts) {
				ellTheta_, _ := ellTheta[itm].Get([]int{i, choices[s]})
				if ellTheta_ < math.SmallestNonzeroFloat64 {
					ellTheta_ = math.SmallestNonzeroFloat64
				}
				lpInfty[i] = math.Log(piAlphat[i]) + math.Log(ellTheta_)
			}
		}

		piInfty := math2.EnergyToDensity(lpInfty, bs.AbilityGridPts)
		// build integrand for each item
		for itm, choices := range samples {
			integrand := make([]float64, len(bs.AbilityGridPts))
			for i := range len(bs.AbilityGridPts) {
				ellTheta_, _ := ellTheta[itm].Get([]int{i, choices[s]})
				if ellTheta_ < math.SmallestNonzeroFloat64 {
					ellTheta_ = math.SmallestNonzeroFloat64
				}
				integrand[i] = piInfty[i] * math.Log(ellTheta_)
			}
			integral1 := math2.Trapz2(integrand, bs.AbilityGridPts)
			secondTerm := math.Log(expectedEll[itm][choices[s]])
			delta := secondTerm - integral1
			crit[itm] += delta / float64(ks.NumSamples)
		}
	}
	return crit
}

func (ks McKlSelector) NextItem(bs *models.BayesianScorer) *models.Item {
	deltaItem := ks.Criterion(bs)
	T := ks.Temperature

	if T == 0 {
		var selected string
		var maxval float64
		for key, value := range deltaItem {
			if value > maxval {
				selected = key
				maxval = value
			}
		}
		return getItemByName(selected, bs.Model.GetItems())
	}

	selectionProbs := make(map[string]float64)
	for key, value := range deltaItem {
		selectionProbs[key] = math.Exp(value / T)
	}

	selected := sample(selectionProbs)
	return getItemByName(selected, bs.Model.GetItems())
}
