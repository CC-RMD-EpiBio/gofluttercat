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

package math

import "math"

// OrdinalCumulativeProbs computes P(Y <= k) = sigmoid(c_k - eta) for each cutpoint.
func OrdinalCumulativeProbs(cutpoints []float64, eta float64) []float64 {
	probs := make([]float64, len(cutpoints))
	for i, c := range cutpoints {
		probs[i] = Sigmoid(c - eta)
	}
	return probs
}

// OrdinalPMF computes category probabilities from cutpoints and linear predictor.
// For K cutpoints, returns K+1 category probabilities.
// P(Y=0) = P(Y<=0), P(Y=k) = P(Y<=k) - P(Y<=k-1), P(Y=K) = 1 - P(Y<=K-1).
func OrdinalPMF(cutpoints []float64, eta float64) []float64 {
	if len(cutpoints) == 0 {
		return []float64{1.0}
	}
	cumProbs := OrdinalCumulativeProbs(cutpoints, eta)
	nCategories := len(cutpoints) + 1
	pmf := make([]float64, nCategories)
	pmf[0] = cumProbs[0]
	for k := 1; k < len(cutpoints); k++ {
		pmf[k] = cumProbs[k] - cumProbs[k-1]
	}
	pmf[nCategories-1] = 1.0 - cumProbs[len(cutpoints)-1]

	// Clip negatives and renormalize to ensure valid distribution
	var total float64
	for i := range pmf {
		if pmf[i] < 0 {
			pmf[i] = 0
		}
		total += pmf[i]
	}
	if total > 0 {
		for i := range pmf {
			pmf[i] /= total
		}
	} else {
		for i := range pmf {
			pmf[i] = 1.0 / float64(nCategories)
		}
	}
	return pmf
}

// OrdinalExpectedValue computes E[Y] = sum(k * P(Y=k)) for an ordinal model.
func OrdinalExpectedValue(cutpoints []float64, eta float64) float64 {
	pmf := OrdinalPMF(cutpoints, eta)
	var ev float64
	for k, p := range pmf {
		ev += float64(k) * p
	}
	return ev
}

// Softmax computes a numerically stable softmax over the input values.
// If all inputs are -Inf or NaN, returns uniform weights.
func Softmax(values []float64) []float64 {
	if len(values) == 0 {
		return nil
	}
	// Find max of finite values for numerical stability
	maxVal := math.Inf(-1)
	for _, v := range values {
		if !math.IsInf(v, 0) && !math.IsNaN(v) && v > maxVal {
			maxVal = v
		}
	}
	// If no finite values, return uniform weights
	if math.IsInf(maxVal, -1) {
		result := make([]float64, len(values))
		uniform := 1.0 / float64(len(values))
		for i := range result {
			result[i] = uniform
		}
		return result
	}
	result := make([]float64, len(values))
	var sum float64
	for i, v := range values {
		if !math.IsInf(v, 0) && !math.IsNaN(v) {
			result[i] = math.Exp(v - maxVal)
		} else {
			result[i] = 0
		}
		sum += result[i]
	}
	for i := range result {
		result[i] /= sum
	}
	return result
}
