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

// VariableType represents the type of a variable in the imputation model.
type VariableType string

const (
	Continuous VariableType = "continuous"
	Binary     VariableType = "binary"
	Ordinal    VariableType = "ordinal"
)

// UnivariateModelResult holds metadata and point estimates for one fitted univariate model.
type UnivariateModelResult struct {
	InterceptMean   *float64
	BetaMean        []float64
	CutpointsMean   []float64
	NObs            int
	ElpdLoo         float64
	ElpdLooPerObs   float64
	ElpdLooPerObsSe float64
	KhatMax         float64
	KhatMean        float64
	PredictorIdx    int // -1 for zero-predictor
	TargetIdx       int
	PredictorMean   float64
	PredictorStd    float64
	Converged       bool
}

// PairwiseStackingModel represents a loaded pairwise ordinal stacking model.
type PairwiseStackingModel struct {
	VariableTypes    map[int]VariableType
	PredictionGraph  map[string][]string
	ZeroPredictors   map[int]*UnivariateModelResult
	UnivariateModels map[[2]int]*UnivariateModelResult // key: [targetIdx, predictorIdx]
	MixedWeights     map[string]float64                // item name → pairwise weight (0-1)
	Version          string
	VariableNames    []string
	NObs             int
}
