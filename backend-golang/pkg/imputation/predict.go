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

package imputation

import (
	"fmt"
	"math"

	catmath "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/math"
)

// Predict computes a stacked point prediction for the target variable given
// observed items. Models are weighted by LOO-CV ELPD with an uncertainty penalty.
func (m *MiceBayesianLoo) Predict(items map[string]float64, target string, uncertaintyPenalty float64) (float64, error) {
	targetIdx := m.VariableIndex(target)
	if targetIdx < 0 {
		return 0, fmt.Errorf("unknown target variable: %s", target)
	}

	varType, ok := m.VariableTypes[targetIdx]
	if !ok {
		varType = Continuous
	}

	// Collect eligible models and their predictions.
	// Per-obs ELPD and SE are used directly — no rescaling by N.
	// Each model's per-obs ELPD reflects its quality per prediction,
	// and per-obs SE reflects the uncertainty in that estimate.
	var predictions []float64
	var elpdValues []float64
	var seValues []float64

	// Zero-predictor model (always available if it exists and converged)
	if zp, ok := m.ZeroPredictors[targetIdx]; ok && zp.Converged {
		pred := predictSingleUnivariate(zp, 0, varType)
		predictions = append(predictions, pred)
		elpdValues = append(elpdValues, zp.ElpdLooPerObs)
		seValues = append(seValues, zp.ElpdLooPerObsSe)
	}

	// Univariate models whose predictor is in items
	for name, value := range items {
		predIdx := m.VariableIndex(name)
		if predIdx < 0 {
			continue
		}
		key := [2]int{targetIdx, predIdx}
		um, ok := m.UnivariateModels[key]
		if !ok || !um.Converged {
			continue
		}
		pred := predictSingleUnivariate(um, value, varType)
		predictions = append(predictions, pred)
		elpdValues = append(elpdValues, um.ElpdLooPerObs)
		seValues = append(seValues, um.ElpdLooPerObsSe)
	}

	if len(predictions) == 0 {
		return 0, fmt.Errorf("no converged models available for target %s", target)
	}

	weights := computeStackingWeights(elpdValues, seValues, uncertaintyPenalty)

	var result float64
	for i, w := range weights {
		result += w * predictions[i]
	}
	return result, nil
}

// PredictPMF computes a stacked ordinal PMF for the target variable.
// Returns a probability distribution over nCategories categories.
func (m *MiceBayesianLoo) PredictPMF(items map[string]float64, target string, nCategories int, uncertaintyPenalty float64) ([]float64, error) {
	targetIdx := m.VariableIndex(target)
	if targetIdx < 0 {
		return nil, fmt.Errorf("unknown target variable: %s", target)
	}

	type modelPMF struct {
		pmf  []float64
		elpd float64
		se   float64
	}

	var models []modelPMF

	// Zero-predictor model
	if zp, ok := m.ZeroPredictors[targetIdx]; ok && zp.Converged {
		pmf := ordinalPMF(zp, 0, nCategories)
		models = append(models, modelPMF{pmf: pmf, elpd: zp.ElpdLooPerObs, se: zp.ElpdLooPerObsSe})
	}

	// Univariate models
	for name, value := range items {
		predIdx := m.VariableIndex(name)
		if predIdx < 0 {
			continue
		}
		key := [2]int{targetIdx, predIdx}
		um, ok := m.UnivariateModels[key]
		if !ok || !um.Converged {
			continue
		}
		pmf := ordinalPMF(um, value, nCategories)
		models = append(models, modelPMF{pmf: pmf, elpd: um.ElpdLooPerObs, se: um.ElpdLooPerObsSe})
	}

	if len(models) == 0 {
		return nil, fmt.Errorf("no converged models available for target %s", target)
	}

	elpdValues := make([]float64, len(models))
	seValues := make([]float64, len(models))
	for i, mod := range models {
		elpdValues[i] = mod.elpd
		seValues[i] = mod.se
	}
	weights := computeStackingWeights(elpdValues, seValues, uncertaintyPenalty)

	// Weighted mixture of PMFs
	result := make([]float64, nCategories)
	for i, w := range weights {
		for k := range nCategories {
			if k < len(models[i].pmf) {
				result[k] += w * models[i].pmf[k]
			}
		}
	}
	return result, nil
}

// PredictChained computes a prediction for target by chaining through the
// prediction graph from source. It finds a path from source to target and
// propagates the value through intermediate predictions.
func (m *MiceBayesianLoo) PredictChained(target, source string, value float64) (float64, error) {
	path := m.FindPredictionPath(target, source)
	if path == nil {
		return 0, fmt.Errorf("no prediction path from %s to %s", source, target)
	}

	currentValue := value
	for i := 0; i < len(path)-1; i++ {
		from := path[i]
		to := path[i+1]

		fromIdx := m.VariableIndex(from)
		toIdx := m.VariableIndex(to)
		if fromIdx < 0 || toIdx < 0 {
			return 0, fmt.Errorf("unknown variable in path: %s -> %s", from, to)
		}

		varType, ok := m.VariableTypes[toIdx]
		if !ok {
			varType = Continuous
		}

		key := [2]int{toIdx, fromIdx}
		um, ok := m.UnivariateModels[key]
		if !ok {
			return 0, fmt.Errorf("no model for %s -> %s", from, to)
		}

		currentValue = predictSingleUnivariate(um, currentValue, varType)
	}
	return currentValue, nil
}

// FindPredictionPath finds a path from source to target in the prediction graph
// using breadth-first search. Returns nil if no path exists.
func (m *MiceBayesianLoo) FindPredictionPath(target, source string) []string {
	if target == source {
		return []string{source}
	}

	// BFS from source to target
	// The prediction graph maps target -> predictors, so we need to
	// build a reverse adjacency: predictor -> targets it can predict
	reverse := make(map[string][]string)
	for tgt, predictors := range m.PredictionGraph {
		for _, pred := range predictors {
			reverse[pred] = append(reverse[pred], tgt)
		}
	}

	visited := map[string]bool{source: true}
	parent := map[string]string{}
	queue := []string{source}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, next := range reverse[current] {
			if visited[next] {
				continue
			}
			visited[next] = true
			parent[next] = current
			if next == target {
				// Reconstruct path
				var path []string
				for node := target; node != source; node = parent[node] {
					path = append([]string{node}, path...)
				}
				path = append([]string{source}, path...)
				return path
			}
			queue = append(queue, next)
		}
	}
	return nil
}

// predictSingleUnivariate computes a point prediction from a single univariate model.
func predictSingleUnivariate(result *UnivariateModelResult, predictorValue float64, varType VariableType) float64 {
	eta := computeEta(result, predictorValue)

	switch varType {
	case Binary:
		return catmath.Sigmoid(eta)
	case Ordinal:
		if len(result.CutpointsMean) > 0 {
			return catmath.OrdinalExpectedValue(result.CutpointsMean, eta)
		}
		return eta
	default: // Continuous
		return eta
	}
}

// ordinalPMF computes the ordinal PMF from a single model.
func ordinalPMF(result *UnivariateModelResult, predictorValue float64, nCategories int) []float64 {
	eta := computeEta(result, predictorValue)

	if len(result.CutpointsMean) > 0 {
		return catmath.OrdinalPMF(result.CutpointsMean, eta)
	}

	// Fallback: degenerate PMF placing all mass on the expected category
	pmf := make([]float64, nCategories)
	if nCategories > 0 {
		idx := max(int(eta+0.5), 0)
		if idx >= nCategories {
			idx = nCategories - 1
		}
		pmf[idx] = 1.0
	}
	return pmf
}

// computeEta computes the linear predictor for a univariate model.
func computeEta(result *UnivariateModelResult, predictorValue float64) float64 {
	var eta float64

	if result.InterceptMean != nil {
		eta = *result.InterceptMean
	}

	if len(result.BetaMean) > 0 && result.PredictorIdx >= 0 {
		// Standardize predictor
		xStd := predictorValue
		if result.PredictorStd > 0 {
			xStd = (predictorValue - result.PredictorMean) / result.PredictorStd
		}
		eta += xStd * result.BetaMean[0]
	}

	return eta
}

// computeStackingWeights computes Bayesian stacking weights from ELPD values.
// w_k = softmax(elpd_k - penalty * se_k)
// Infinite SE values are clamped to 1e6 for numerical safety.
func computeStackingWeights(elpdValues, seValues []float64, penalty float64) []float64 {
	scores := make([]float64, len(elpdValues))
	for i := range elpdValues {
		se := seValues[i]
		if math.IsInf(se, 0) || math.IsNaN(se) {
			se = 1e6
		}
		scores[i] = elpdValues[i] - penalty*se
	}
	return catmath.Softmax(scores)
}