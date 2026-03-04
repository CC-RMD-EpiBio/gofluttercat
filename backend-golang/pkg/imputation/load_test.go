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
	"math"
	"testing"
)

const testModelDir = "testdata/test_model"
const testModelSafetensorsDir = "testdata/test_model_safetensors"

func Test_LoadFromDisk(t *testing.T) {
	model, err := LoadFromDisk(testModelDir)
	if err != nil {
		t.Fatalf("LoadFromDisk failed: %v", err)
	}

	if model.Version != "2.0" {
		t.Errorf("expected version 2.0, got %s", model.Version)
	}
	if model.NObs != 100 {
		t.Errorf("expected 100 obs, got %d", model.NObs)
	}
	if len(model.VariableNames) != 3 {
		t.Errorf("expected 3 variables, got %d", len(model.VariableNames))
	}
	if model.VariableNames[0] != "age" || model.VariableNames[1] != "pain" || model.VariableNames[2] != "mobility" {
		t.Errorf("unexpected variable names: %v", model.VariableNames)
	}

	// Check variable types
	if model.VariableTypes[0] != Continuous {
		t.Errorf("expected continuous for idx 0, got %s", model.VariableTypes[0])
	}
	if model.VariableTypes[1] != Ordinal {
		t.Errorf("expected ordinal for idx 1, got %s", model.VariableTypes[1])
	}

	// Check zero-predictor models loaded
	if len(model.ZeroPredictors) != 3 {
		t.Errorf("expected 3 zero-predictor models, got %d", len(model.ZeroPredictors))
	}
	zp1 := model.ZeroPredictors[1]
	if zp1 == nil {
		t.Fatal("zero-predictor for pain (idx 1) not found")
	}
	if zp1.PredictorIdx != -1 {
		t.Errorf("expected predictor_idx -1 for zero-predictor, got %d", zp1.PredictorIdx)
	}
	if !zp1.Converged {
		t.Error("expected zero-predictor to be converged")
	}
	if len(zp1.CutpointsMean) != 4 {
		t.Errorf("expected 4 cutpoints, got %d", len(zp1.CutpointsMean))
	}

	// Check univariate models loaded
	if len(model.UnivariateModels) != 3 {
		t.Errorf("expected 3 univariate models, got %d", len(model.UnivariateModels))
	}
	um := model.UnivariateModels[[2]int{1, 0}]
	if um == nil {
		t.Fatal("univariate model [1,0] (age->pain) not found")
	}
	if len(um.BetaMean) != 1 || um.BetaMean[0] != 0.5 {
		t.Errorf("unexpected beta_mean: %v", um.BetaMean)
	}
	if um.PredictorMean != 50.0 {
		t.Errorf("expected predictor_mean 50.0, got %f", um.PredictorMean)
	}

	// Check prediction graph
	if len(model.PredictionGraph["pain"]) != 1 || model.PredictionGraph["pain"][0] != "age" {
		t.Errorf("unexpected prediction graph for pain: %v", model.PredictionGraph["pain"])
	}
	if len(model.PredictionGraph["mobility"]) != 2 {
		t.Errorf("expected 2 predictors for mobility, got %d", len(model.PredictionGraph["mobility"]))
	}
}

func Test_Predict(t *testing.T) {
	model, err := LoadFromDisk(testModelDir)
	if err != nil {
		t.Fatalf("LoadFromDisk failed: %v", err)
	}

	// Predict pain given age=60, penalty=0
	// Per-obs ELPD is scaled by reference N before softmax (fair comparison across models)
	items := map[string]float64{"age": 60.0}
	pred, err := model.Predict(items, "pain", 0.0)
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}

	expected := 2.233047
	if math.Abs(pred-expected) > 1e-4 {
		t.Errorf("prediction mismatch: got %f, expected %f", pred, expected)
	}
}

func Test_PredictPMF(t *testing.T) {
	model, err := LoadFromDisk(testModelDir)
	if err != nil {
		t.Fatalf("LoadFromDisk failed: %v", err)
	}

	// Predict PMF for pain given age=60, penalty=0, 5 categories
	items := map[string]float64{"age": 60.0}
	pmf, err := model.PredictPMF(items, "pain", 5, 0.0)
	if err != nil {
		t.Fatalf("PredictPMF failed: %v", err)
	}

	if len(pmf) != 5 {
		t.Fatalf("expected 5 categories, got %d", len(pmf))
	}

	// Check PMF sums to 1
	var sum float64
	for _, p := range pmf {
		sum += p
	}
	if math.Abs(sum-1.0) > 1e-6 {
		t.Errorf("PMF sum = %f, expected 1.0", sum)
	}

	// Check individual values match per-obs ELPD weighted output
	expectedPMF := []float64{0.144123, 0.167298, 0.235906, 0.216756, 0.235918}
	for i, p := range pmf {
		if math.Abs(p-expectedPMF[i]) > 1e-4 {
			t.Errorf("PMF[%d] = %f, expected %f", i, p, expectedPMF[i])
		}
	}
}

func Test_FindPredictionPath(t *testing.T) {
	model, err := LoadFromDisk(testModelDir)
	if err != nil {
		t.Fatalf("LoadFromDisk failed: %v", err)
	}

	// Direct path: age -> pain
	path := model.FindPredictionPath("pain", "age")
	if len(path) != 2 || path[0] != "age" || path[1] != "pain" {
		t.Errorf("unexpected path age->pain: %v", path)
	}

	// Direct path: age -> mobility
	path = model.FindPredictionPath("mobility", "age")
	if len(path) != 2 || path[0] != "age" || path[1] != "mobility" {
		t.Errorf("unexpected path age->mobility: %v", path)
	}

	// Chained path: pain -> mobility (pain is a predictor of mobility)
	path = model.FindPredictionPath("mobility", "pain")
	if len(path) != 2 || path[0] != "pain" || path[1] != "mobility" {
		t.Errorf("unexpected path pain->mobility: %v", path)
	}

	// Same variable
	path = model.FindPredictionPath("age", "age")
	if len(path) != 1 || path[0] != "age" {
		t.Errorf("unexpected self-path: %v", path)
	}

	// No path
	path = model.FindPredictionPath("age", "pain")
	if path != nil {
		t.Errorf("expected nil path for pain->age, got %v", path)
	}
}

func Test_LoadFromDisk_Safetensors(t *testing.T) {
	model, err := LoadFromDisk(testModelSafetensorsDir)
	if err != nil {
		t.Fatalf("LoadFromDisk (safetensors) failed: %v", err)
	}

	if model.Version != "2.0" {
		t.Errorf("expected version 2.0, got %s", model.Version)
	}
	if model.NObs != 100 {
		t.Errorf("expected 100 obs, got %d", model.NObs)
	}
	if len(model.VariableNames) != 3 {
		t.Errorf("expected 3 variables, got %d", len(model.VariableNames))
	}

	// Check variable types
	if model.VariableTypes[0] != Continuous {
		t.Errorf("expected continuous for idx 0, got %s", model.VariableTypes[0])
	}
	if model.VariableTypes[1] != Ordinal {
		t.Errorf("expected ordinal for idx 1, got %s", model.VariableTypes[1])
	}

	// Check zero-predictor models loaded
	if len(model.ZeroPredictors) != 3 {
		t.Errorf("expected 3 zero-predictor models, got %d", len(model.ZeroPredictors))
	}
	zp1 := model.ZeroPredictors[1]
	if zp1 == nil {
		t.Fatal("zero-predictor for pain (idx 1) not found")
	}
	if zp1.PredictorIdx != -1 {
		t.Errorf("expected predictor_idx -1 for zero-predictor, got %d", zp1.PredictorIdx)
	}
	if len(zp1.CutpointsMean) != 4 {
		t.Errorf("expected 4 cutpoints, got %d", len(zp1.CutpointsMean))
	}

	// Check univariate models loaded
	if len(model.UnivariateModels) != 3 {
		t.Errorf("expected 3 univariate models, got %d", len(model.UnivariateModels))
	}
	um := model.UnivariateModels[[2]int{1, 0}]
	if um == nil {
		t.Fatal("univariate model [1,0] (age->pain) not found")
	}
	if len(um.BetaMean) != 1 || um.BetaMean[0] != 0.5 {
		t.Errorf("unexpected beta_mean: %v", um.BetaMean)
	}
	if um.PredictorMean != 50.0 {
		t.Errorf("expected predictor_mean 50.0, got %f", um.PredictorMean)
	}
}

func Test_Predict_Safetensors(t *testing.T) {
	model, err := LoadFromDisk(testModelSafetensorsDir)
	if err != nil {
		t.Fatalf("LoadFromDisk (safetensors) failed: %v", err)
	}

	// Same prediction test as HDF5 version (per-obs ELPD weighted)
	items := map[string]float64{"age": 60.0}
	pred, err := model.Predict(items, "pain", 0.0)
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}

	expected := 2.233047
	if math.Abs(pred-expected) > 1e-4 {
		t.Errorf("prediction mismatch: got %f, expected %f", pred, expected)
	}
}

func Test_PredictPMF_Safetensors(t *testing.T) {
	model, err := LoadFromDisk(testModelSafetensorsDir)
	if err != nil {
		t.Fatalf("LoadFromDisk (safetensors) failed: %v", err)
	}

	items := map[string]float64{"age": 60.0}
	pmf, err := model.PredictPMF(items, "pain", 5, 0.0)
	if err != nil {
		t.Fatalf("PredictPMF failed: %v", err)
	}

	if len(pmf) != 5 {
		t.Fatalf("expected 5 categories, got %d", len(pmf))
	}

	var sum float64
	for _, p := range pmf {
		sum += p
	}
	if math.Abs(sum-1.0) > 1e-6 {
		t.Errorf("PMF sum = %f, expected 1.0", sum)
	}

	expectedPMF := []float64{0.144123, 0.167298, 0.235906, 0.216756, 0.235918}
	for i, p := range pmf {
		if math.Abs(p-expectedPMF[i]) > 1e-4 {
			t.Errorf("PMF[%d] = %f, expected %f", i, p, expectedPMF[i])
		}
	}
}

func Test_PredictChained(t *testing.T) {
	model, err := LoadFromDisk(testModelDir)
	if err != nil {
		t.Fatalf("LoadFromDisk failed: %v", err)
	}

	// Chained prediction: age=60 -> pain -> mobility
	// First: age->pain: x_std=(60-50)/15=0.6667, eta=0.6667*0.5+0.2=0.5333
	// pain E[Y] with cutpoints [-1.5,-0.5,0.5,1.5], eta=0.5333 -> ~2.4057
	// Then: pain->mobility: x_std=(2.4057-2.0)/1.0=0.4057, eta=0.4057*0.8+0.1=0.4246
	// mobility E[Y] with cutpoints [-2.0,-0.8,0.8,2.0], eta=0.4246
	pred, err := model.PredictChained("mobility", "age", 60.0)
	if err != nil {
		t.Fatalf("PredictChained failed: %v", err)
	}

	// The result should be a reasonable ordinal expected value (between 0 and 4)
	if pred < 0 || pred > 4 {
		t.Errorf("chained prediction out of range: %f", pred)
	}
}
