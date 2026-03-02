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
	"math/rand"
	"slices"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/imputation"
	math2 "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/math"
	"github.com/mederrata/ndvek"
	"github.com/sgreben/piecewiselinear"
	"github.com/viterin/vek"
)

type Score interface {
	Mean() *ndvek.NdArray
	Std() *ndvek.NdArray
}

type Responses struct {
	Responses []Response
}

type Scorer interface {
	Score(*IrtModel, *Responses) error
}
type BayesianScore struct {
	Energy   []float64
	Grid     []float64
	RbEnergy []float64
}

func DefaultAbilityPrior(x float64) float64 {
	m := math2.NewGaussianDistribution(0, 2)
	return m.Density(x)
}

type BayesianScorer struct {
	AbilityGridPts  []float64
	Prior           func(float64) float64
	Model           IrtModel
	Answered        []*Response
	Scored          map[string]int
	Running         *BayesianScore
	Exclusions      []string
	ImputationModel *imputation.MiceBayesianLoo
}

func (bs BayesianScore) Sample(numSamples int) []float64 {
	samples := make([]float64, numSamples)
	density := bs.Density()
	cum := vek.CumSum(density)
	cum = vek.DivNumber(cum, cum[len(cum)-1])
	f := piecewiselinear.Function{Y: bs.Grid}
	f.X = cum
	for n := range numSamples {
		r := rand.Float64()
		samples[n] = f.At(r)
	}
	return samples
}

func (bs *BayesianScorer) Score(resp *Responses) error {
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

	bs.Running.RbEnergy = bs.ScoreRaoBlackwell()

	return nil
}

// ScoreRaoBlackwell computes the Rao-Blackwellized posterior energy by
// marginalizing over unobserved items using the MICEBayesianLoo imputation
// model. For each admissible item, the imputation model provides a PMF
// over response categories conditioned on the observed responses.
//
// The marginalization is done correctly under the log:
//
//	log pi_RB(theta) = log pi_t(theta) + sum_j log[ sum_k q(k) * P(Y_j=k | theta) ]
//
// This computes log E_q[P(Y|theta)] (marginalize likelihood, then log),
// matching bayesianquilts' analytic Rao-Blackwellization:
//
//	rb = logsumexp_k( log q(k) + log P(Y=k|theta) )
func (bs BayesianScorer) ScoreRaoBlackwell() []float64 {
	if bs.ImputationModel == nil {
		out := make([]float64, len(bs.Running.Energy))
		copy(out, bs.Running.Energy)
		return out
	}

	admissible := AdmissibleItems(&bs)
	abilities, err := ndvek.NewNdArray([]int{len(bs.AbilityGridPts)}, bs.AbilityGridPts)
	if err != nil {
		panic(err)
	}

	probs := bs.Model.Prob(abilities)

	// Build observed items map from answered responses
	observedItems := make(map[string]float64)
	for _, r := range bs.Answered {
		observedItems[r.Item.Name] = float64(r.Value)
	}

	lpi_t := bs.Running.Energy
	lpi_z := make([]float64, len(bs.AbilityGridPts))
	nGrid := len(bs.AbilityGridPts)

	for _, itm := range admissible {
		p := probs[itm.Name]
		K := p.Shape()[1]

		// Get PMF from MICEBayesianLoo model
		pmf, err := bs.ImputationModel.PredictPMF(observedItems, itm.Name, K, 0.0)
		if err != nil {
			continue // skip items the imputation model doesn't know about
		}

		// Precompute log q(k) for the imputation PMF
		logQ := make([]float64, K)
		for k := range K {
			if pmf[k] > 0 {
				logQ[k] = math.Log(pmf[k])
			} else {
				logQ[k] = math.Inf(-1)
			}
		}

		// For each grid point, compute log[ sum_k q(k) * p(Y=k|theta_i) ]
		// = logsumexp_k( log q(k) + log p(Y=k|theta_i) )
		for i := range nGrid {
			terms := make([]float64, K)
			for k := range K {
				pp, _ := p.Get([]int{i, k})
				if pp < 1e-30 {
					pp = 1e-30
				}
				terms[k] = logQ[k] + math.Log(pp)
			}
			lpi_z[i] += math2.LogSumExp(terms)
		}
	}

	return vek.Add(lpi_t, lpi_z)
}

func (bs *BayesianScorer) AddResponses(resp []Response) error {
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

func (bs *BayesianScorer) RemoveResponses(itmNames []string) error {
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

func NewBayesianScorer(AbilityGridPts []float64, abilityPrior func(float64) float64, model IrtModel) *BayesianScorer {

	// initialize the prior
	energy := make([]float64, 0)
	for _, x := range AbilityGridPts {
		density := abilityPrior(x)
		energy = append(energy, math.Log(density))
	}

	bs := &BayesianScorer{
		AbilityGridPts: AbilityGridPts,
		Prior:          abilityPrior,
		Model:          model,
		Running: &BayesianScore{
			Grid:     AbilityGridPts,
			Energy:   energy,
			RbEnergy: energy,
		},
	}
	bs.Running.RbEnergy = bs.ScoreRaoBlackwell()
	return bs
}

func (bs BayesianScore) Density() []float64 {
	d := math2.EnergyToDensity(bs.Energy, bs.Grid)
	return d
}

func (bs BayesianScore) Mean() float64 {
	d := bs.Density()
	mean := math2.Trapz2(vek.Mul(d, bs.Grid), bs.Grid)
	return mean
}

func (bs BayesianScore) Std() float64 {
	d := bs.Density()
	mean := math2.Trapz2(vek.Mul(d, bs.Grid), bs.Grid)
	second := math2.Trapz2(vek.Mul(bs.Grid, vek.Mul(d, bs.Grid)), bs.Grid)
	return math.Sqrt(second - mean*mean)
}

func (bs BayesianScore) Deciles() []float64 {
	density := math2.EnergyToDensity(bs.Energy, bs.Grid)
	cum := vek.CumSum(density)
	cum = vek.DivNumber(cum, cum[len(cum)-1])
	f := piecewiselinear.Function{Y: bs.Grid}
	f.X = cum
	deciles := make([]float64, 0)
	for r := range 9 {
		deciles = append(deciles, f.At((float64(r)+1)/10))
	}
	return deciles
}

func (bs BayesianScore) RbDensity() []float64 {
	d := math2.EnergyToDensity(bs.RbEnergy, bs.Grid)
	return d
}

func (bs BayesianScore) RbMean() float64 {
	d := bs.RbDensity()
	mean := math2.Trapz2(vek.Mul(d, bs.Grid), bs.Grid)
	return mean
}

func (bs BayesianScore) RbStd() float64 {
	d := bs.RbDensity()
	mean := math2.Trapz2(vek.Mul(d, bs.Grid), bs.Grid)
	second := math2.Trapz2(vek.Mul(bs.Grid, vek.Mul(d, bs.Grid)), bs.Grid)
	return math.Sqrt(second - mean*mean)
}

func (bs BayesianScore) RbDeciles() []float64 {
	density := bs.RbDensity()
	cum := vek.CumSum(density)
	cum = vek.DivNumber(cum, cum[len(cum)-1])
	f := piecewiselinear.Function{Y: bs.Grid}
	f.X = cum
	deciles := make([]float64, 0)
	for r := range 9 {
		deciles = append(deciles, f.At((float64(r)+1)/10))
	}
	return deciles
}
