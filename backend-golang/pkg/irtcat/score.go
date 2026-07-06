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

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/biascorrection"
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
	m := math2.NewGaussianDistribution(0, 2.5)
	return m.Density(x)
}

type BayesianScorer struct {
	Model           IrtModel
	ImputationModel imputation.ImputationModel
	Prior           func(float64) float64
	Scored          map[string]int
	Running         *BayesianScore
	// BaselineScorer is the scorer for the baseline IRT model. When set,
	// it is scored first so that the mixed imputation model can use
	// the baseline posterior to compute IRT-based PMFs.
	BaselineScorer *BayesianScorer
	AbilityGridPts []float64
	Answered       []*Response
	Exclusions     []string
	// Bcm, when non-nil, is the conditional bias-correction map for this
	// scale. AdjustedScore applies it to the observed-only EAP.
	Bcm *biascorrection.BCMConditional
}

// DefaultEnergy returns RbEnergy (marginalized) when available, otherwise Energy.
func (bs BayesianScore) DefaultEnergy() []float64 {
	if len(bs.RbEnergy) > 0 {
		return bs.RbEnergy
	}
	return bs.Energy
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
	// Score baseline first so that the mixed imputation model can use
	// the baseline posterior for its IRT PMFs.
	if bs.BaselineScorer != nil {
		if err := bs.BaselineScorer.Score(resp); err != nil {
			return err
		}
	}

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
// marginalizing over unobserved items using the PairwiseStackingModel imputation
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

	// Wire up the baseline scorer to the mixed imputation model so it
	// can generate IRT-based PMFs from the baseline posterior.
	if mixed, ok := bs.ImputationModel.(*imputation.IrtMixedImputationModel); ok && bs.BaselineScorer != nil {
		mixed.Baseline = bs.BaselineScorer
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

		// Get PMF from PairwiseStackingModel model
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

	// Propagate to baseline scorer so its posterior stays current.
	if bs.BaselineScorer != nil {
		if err := bs.BaselineScorer.AddResponses(resp); err != nil {
			return err
		}
	}

	abilities, err := ndvek.NewNdArray([]int{len(bs.AbilityGridPts)}, bs.AbilityGridPts)
	if err != nil {
		panic(err)
	}

	ll := bs.Model.LogLikelihood(abilities, resp)
	for _, r := range resp {
		bs.Answered = append(bs.Answered, &r)
	}
	bs.Running.Energy = vek.Add(bs.Running.Energy, ll.Float64Data())

	return nil
}

func (bs *BayesianScorer) RemoveResponses(itmNames []string) error {
	// Propagate to baseline scorer.
	if bs.BaselineScorer != nil {
		if err := bs.BaselineScorer.RemoveResponses(itmNames); err != nil {
			return err
		}
	}

	toDelete := make([]Response, 0)
	toDeleteNames := make([]string, 0)
	for _, r := range bs.Answered {
		if slices.Contains(itmNames, r.Item.Name) {
			toDelete = append(toDelete, *r)
			toDeleteNames = append(toDeleteNames, r.Item.Name)
		}
	}
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
	bs.Running.Energy = vek.Sub(bs.Running.Energy, ll.Float64Data())

	return nil
}

// BaselinePMF returns a marginalized PMF for the target item by integrating
// P(Y=k|theta) over the current posterior. This implements the
// imputation.BaselinePredictor interface.
func (bs *BayesianScorer) BaselinePMF(target string, nCategories int) ([]float64, error) {
	abilities, err := ndvek.NewNdArray([]int{len(bs.AbilityGridPts)}, bs.AbilityGridPts)
	if err != nil {
		return nil, err
	}

	probs := bs.Model.Prob(abilities)
	p, ok := probs[target]
	if !ok {
		return nil, fmt.Errorf("item %q not found in baseline model", target)
	}

	K := p.Shape()[1]
	if K != nCategories {
		return nil, fmt.Errorf("baseline model has %d categories for %q, expected %d", K, target, nCategories)
	}

	// Get posterior density from the current baseline energy
	density := math2.EnergyToDensity(bs.Running.Energy, bs.AbilityGridPts)

	// Marginalize: P(Y=k) = integral density(theta) * P(Y=k|theta) dtheta
	pmf := make([]float64, nCategories)
	nGrid := len(bs.AbilityGridPts)
	for k := range nCategories {
		integrand := make([]float64, nGrid)
		for i := range nGrid {
			pp, _ := p.Get([]int{i, k})
			integrand[i] = density[i] * pp
		}
		pmf[k] = math2.Trapz2(integrand, bs.AbilityGridPts)
	}

	// Normalize
	var total float64
	for k := range nCategories {
		total += pmf[k]
	}
	if total > 0 {
		for k := range nCategories {
			pmf[k] /= total
		}
	}
	return pmf, nil
}

// NewBayesianScorer builds a scorer for `model`. When `bcm` is non-nil it is
// attached so AdjustedScore returns the bias-corrected EAP; pass nil for no
// correction.
func NewBayesianScorer(AbilityGridPts []float64, abilityPrior func(float64) float64, model IrtModel, bcm *biascorrection.BCMConditional) *BayesianScorer {

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
		Bcm:            bcm,
		Running: &BayesianScore{
			Grid:     AbilityGridPts,
			Energy:   energy,
			RbEnergy: energy,
		},
	}
	bs.Running.RbEnergy = bs.ScoreRaoBlackwell()
	return bs
}

// AdjustedScore returns the bias-corrected EAP for the observed subset. When a
// BCM is attached (bs.Bcm != nil) it applies the conditional correction keyed
// on the administered items; otherwise it returns the plain observed-only EAP
// unchanged. The raw EAP is the observed-only posterior mean — the regime the
// BCM was trained on (no imputation) — regardless of any imputation model.
func (bs *BayesianScorer) AdjustedScore() float64 {
	return bs.Running.AdjustedMean(bs)
}

// NewBayesianScorerWithBaseline creates a scorer and, when the imputation
// model is an IrtMixedImputationModel and a baseline model is provided,
// creates a separate baseline scorer so that the mixed model can query the
// baseline posterior for IRT-based PMFs. The baselineModel has different
// item parameters from the marginalized model.
func NewBayesianScorerWithBaseline(gridPts []float64, prior func(float64) float64, model IrtModel, im imputation.ImputationModel, baselineModel IrtModel) *BayesianScorer {
	scorer := NewBayesianScorer(gridPts, prior, model, nil)
	scorer.ImputationModel = im

	if _, ok := im.(*imputation.IrtMixedImputationModel); ok && baselineModel != nil {
		baseline := NewBayesianScorer(gridPts, prior, baselineModel, nil)
		scorer.BaselineScorer = baseline
	}
	return scorer
}

// Density returns the posterior density using the default energy
// (marginalized when available).
func (bs BayesianScore) Density() []float64 {
	return math2.EnergyToDensity(bs.DefaultEnergy(), bs.Grid)
}

// Mean returns the posterior mean using the default energy.
func (bs BayesianScore) Mean() float64 {
	d := bs.Density()
	return math2.Trapz2(vek.Mul(d, bs.Grid), bs.Grid)
}

// Std returns the posterior standard deviation using the default energy.
func (bs BayesianScore) Std() float64 {
	d := bs.Density()
	mean := math2.Trapz2(vek.Mul(d, bs.Grid), bs.Grid)
	second := math2.Trapz2(vek.Mul(bs.Grid, vek.Mul(d, bs.Grid)), bs.Grid)
	return math.Sqrt(second - mean*mean)
}

// Deciles returns the posterior deciles using the default energy.
func (bs BayesianScore) Deciles() []float64 {
	density := bs.Density()
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

// ObservedOnlyDensity returns density from observed items only (ignoring imputation).
func (bs BayesianScore) ObservedOnlyDensity() []float64 {
	return math2.EnergyToDensity(bs.Energy, bs.Grid)
}

// ObservedOnlyMean returns the posterior mean from observed items only.
func (bs BayesianScore) ObservedOnlyMean() float64 {
	d := bs.ObservedOnlyDensity()
	return math2.Trapz2(vek.Mul(d, bs.Grid), bs.Grid)
}

// AdjustedMean returns THIS score's observed-only posterior mean corrected by
// the scorer's conditional BCM, keyed on the scorer's administered items. When
// the scorer is nil or carries no BCM, the uncorrected observed-only mean is
// returned, so callers don't need to branch.
//
// The correction is the full ApplyByKey map (raw subset EAP -> estimated
// full-bank score) — the exact quantity benchmarked in cat_optimalcontrol,
// where BCM_cond is the best deployable corrector. Because the BCM is trained
// with the full battery and near-full subsets included in its triples (see
// build_bcm_triples), the all-indicators-on row is anchored to identity, so at
// full administration ApplyByKey returns ~raw (zero bias by calibration) and
// the correction vanishes as the instrument is completed — no runtime baseline
// subtraction needed.
//
// The mean comes from this BayesianScore's own energy, while the BCM and the
// administered-item set are read from the scorer — the useful case being a
// standalone score (e.g. a summary snapshot) adjusted with its scorer context.
func (bs BayesianScore) AdjustedMean(scorer *BayesianScorer) float64 {
	raw := bs.ObservedOnlyMean()
	if scorer == nil || scorer.Bcm == nil {
		return raw
	}
	administered := make(map[string]bool, len(scorer.Answered))
	for _, r := range scorer.Answered {
		administered[r.Item.Name] = true
	}
	return scorer.Bcm.ApplyByKey(raw, administered)
}

// AdjustedStd returns THIS score's observed-only posterior SD widened by the
// bias-correction uncertainty due to unadministered items:
//
//	sqrt(error^2 + bias^2)
//
// where error is the observed-only posterior SD and bias is the scorer's
// conditional-BCM incompleteness bias (the correction relative to full
// administration), keyed on the scorer's administered items. The bias term
// vanishes once every item has been administered, so a complete instrument
// reports its plain posterior SD. When the scorer is nil or carries no BCM, the
// uncorrected observed-only SD is returned, so callers don't need to branch.
func (bs BayesianScore) AdjustedStd(scorer *BayesianScorer) float64 {
	errSD := bs.ObservedOnlyStd()
	if scorer == nil || scorer.Bcm == nil {
		return errSD
	}
	administered := make(map[string]bool, len(scorer.Answered))
	for _, r := range scorer.Answered {
		administered[r.Item.Name] = true
	}
	bias := scorer.Bcm.IncompletenessBias(bs.ObservedOnlyMean(), administered)
	return math.Sqrt(errSD*errSD + bias*bias)
}

// ObservedOnlyStd returns the posterior standard deviation from observed items only.
func (bs BayesianScore) ObservedOnlyStd() float64 {
	d := bs.ObservedOnlyDensity()
	mean := math2.Trapz2(vek.Mul(d, bs.Grid), bs.Grid)
	second := math2.Trapz2(vek.Mul(bs.Grid, vek.Mul(d, bs.Grid)), bs.Grid)
	return math.Sqrt(second - mean*mean)
}

// ObservedOnlyDeciles returns posterior deciles from observed items only.
func (bs BayesianScore) ObservedOnlyDeciles() []float64 {
	density := bs.ObservedOnlyDensity()
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

// RbDensity explicitly returns the Rao-Blackwellized density.
func (bs BayesianScore) RbDensity() []float64 {
	return math2.EnergyToDensity(bs.RbEnergy, bs.Grid)
}

// RbMean explicitly returns the Rao-Blackwellized posterior mean.
func (bs BayesianScore) RbMean() float64 {
	d := bs.RbDensity()
	return math2.Trapz2(vek.Mul(d, bs.Grid), bs.Grid)
}

// RbStd explicitly returns the Rao-Blackwellized posterior standard deviation.
func (bs BayesianScore) RbStd() float64 {
	d := bs.RbDensity()
	mean := math2.Trapz2(vek.Mul(d, bs.Grid), bs.Grid)
	second := math2.Trapz2(vek.Mul(bs.Grid, vek.Mul(d, bs.Grid)), bs.Grid)
	return math.Sqrt(second - mean*mean)
}

// RbDeciles explicitly returns the Rao-Blackwellized posterior deciles.
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
