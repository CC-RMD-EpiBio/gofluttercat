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
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/scigolib/hdf5"
	"gopkg.in/yaml.v3"
)

// configYAML mirrors the structure of config.yaml produced by save_to_disk.
type configYAML struct {
	Backend           string                               `yaml:"_backend"`
	PredictionGraph   map[string][]string                  `yaml:"prediction_graph"`
	ZeroPredictorMeta map[string]univariateModelResultYAML `yaml:"zero_predictor_meta"`
	Version           string                               `yaml:"version"`
	UnivariateMeta    []univariateModelResultYAML          `yaml:"univariate_meta"`
	MixedWeights      map[string]float64                   `yaml:"mixed_weights,omitempty"`
	Data              struct {
		VariableTypes map[int]string `yaml:"variable_types"`
		VariableNames []string       `yaml:"variable_names"`
		NObsTotal     int            `yaml:"n_obs_total"`
	} `yaml:"data"`
}

type univariateModelResultYAML struct {
	PredictorIdx    *int      `yaml:"predictor_idx"`
	PredictorMean   *float64  `yaml:"predictor_mean"`
	PredictorStd    *float64  `yaml:"predictor_std"`
	BetaMean        []float64 `yaml:"beta_mean,omitempty"`
	InterceptMean   []float64 `yaml:"intercept_mean,omitempty"`
	CutpointsMean   []float64 `yaml:"cutpoints_mean,omitempty"`
	NObs            int       `yaml:"n_obs"`
	ElpdLoo         float64   `yaml:"elpd_loo"`
	ElpdLooPerObs   float64   `yaml:"elpd_loo_per_obs"`
	ElpdLooPerObsSe float64   `yaml:"elpd_loo_per_obs_se"`
	KhatMax         float64   `yaml:"khat_max"`
	KhatMean        float64   `yaml:"khat_mean"`
	TargetIdx       int       `yaml:"target_idx"`
	Converged       bool      `yaml:"converged"`
}

// LoadFromDisk loads a MiceBayesianLoo model from a directory containing
// config.yaml and either params.h5 or tensors.safetensors.
// The backend is auto-detected from the _backend key in config.yaml;
// defaults to "hdf5" when the key is absent.
func LoadFromDisk(dirPath string) (*MiceBayesianLoo, error) {
	// 1. Read and parse config.yaml
	yamlPath := filepath.Join(dirPath, "config.yaml")
	yamlData, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("reading config.yaml: %w", err)
	}

	var cfg configYAML
	if err := yaml.Unmarshal(yamlData, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config.yaml: %w", err)
	}

	backend := cfg.Backend
	if backend == "" {
		backend = "hdf5"
	}

	// 2. Load numerical arrays from the appropriate backend
	var tensorLookup func(name string) []float64

	switch backend {
	case "hdf5":
		lookup, err := loadTensorsHDF5(dirPath)
		if err != nil {
			return nil, err
		}
		tensorLookup = lookup
	case "safetensors":
		lookup, err := loadTensorsSafetensors(dirPath)
		if err != nil {
			return nil, err
		}
		tensorLookup = lookup
	default:
		return nil, fmt.Errorf("unsupported backend %q in config.yaml", backend)
	}

	// 3. Build variable types map
	varTypes := make(map[int]VariableType, len(cfg.Data.VariableTypes))
	for idx, vt := range cfg.Data.VariableTypes {
		varTypes[idx] = VariableType(vt)
	}

	// 4. Build zero-predictor models
	zeroPredictors := make(map[int]*UnivariateModelResult, len(cfg.ZeroPredictorMeta))
	for key, meta := range cfg.ZeroPredictorMeta {
		targetIdx, err := strconv.Atoi(key)
		if err != nil {
			return nil, fmt.Errorf("parsing zero_predictor key %q: %w", key, err)
		}

		result := metaToResult(meta)
		result.PredictorIdx = -1
		result.TargetIdx = targetIdx

		prefix := fmt.Sprintf("zero_predictor/%d/", targetIdx)
		loadParamsFromLookup(tensorLookup, prefix, result)

		zeroPredictors[targetIdx] = result
	}

	// 5. Build univariate models
	univariateModels := make(map[[2]int]*UnivariateModelResult, len(cfg.UnivariateMeta))
	for _, meta := range cfg.UnivariateMeta {
		result := metaToResult(meta)

		predictorIdx := 0
		if meta.PredictorIdx != nil {
			predictorIdx = *meta.PredictorIdx
		}
		result.PredictorIdx = predictorIdx
		result.TargetIdx = meta.TargetIdx

		prefix := fmt.Sprintf("univariate/%d_%d/", meta.TargetIdx, predictorIdx)
		loadParamsFromLookup(tensorLookup, prefix, result)

		key := [2]int{meta.TargetIdx, predictorIdx}
		univariateModels[key] = result
	}

	return &MiceBayesianLoo{
		Version:          cfg.Version,
		VariableNames:    cfg.Data.VariableNames,
		VariableTypes:    varTypes,
		NObs:             cfg.Data.NObsTotal,
		PredictionGraph:  cfg.PredictionGraph,
		ZeroPredictors:   zeroPredictors,
		UnivariateModels: univariateModels,
		MixedWeights:     cfg.MixedWeights,
	}, nil
}

// loadTensorsHDF5 opens params.h5 and returns a lookup function for tensors by path.
func loadTensorsHDF5(dirPath string) (func(string) []float64, error) {
	h5Path := filepath.Join(dirPath, "params.h5")
	h5File, err := hdf5.Open(h5Path)
	if err != nil {
		return nil, fmt.Errorf("opening params.h5: %w", err)
	}
	defer h5File.Close()

	// Index all HDF5 datasets by path and read their data eagerly.
	tensors := make(map[string][]float64)
	h5File.Walk(func(path string, obj hdf5.Object) {
		if ds, ok := obj.(*hdf5.Dataset); ok {
			data, readErr := ds.Read()
			if readErr == nil {
				// Normalize path: strip leading slash for consistency
				key := strings.TrimPrefix(path, "/")
				tensors[key] = data
			}
		}
	})

	return func(name string) []float64 {
		return tensors[name]
	}, nil
}

// loadTensorsSafetensors reads tensors.safetensors and returns a lookup function.
func loadTensorsSafetensors(dirPath string) (func(string) []float64, error) {
	stPath := filepath.Join(dirPath, "tensors.safetensors")
	tensors, err := readSafetensors(stPath)
	if err != nil {
		return nil, fmt.Errorf("reading tensors.safetensors: %w", err)
	}

	return func(name string) []float64 {
		return tensors[name]
	}, nil
}

func metaToResult(meta univariateModelResultYAML) *UnivariateModelResult {
	result := &UnivariateModelResult{
		NObs:            meta.NObs,
		ElpdLoo:         meta.ElpdLoo,
		ElpdLooPerObs:   meta.ElpdLooPerObs,
		ElpdLooPerObsSe: meta.ElpdLooPerObsSe,
		KhatMax:         meta.KhatMax,
		KhatMean:        meta.KhatMean,
		Converged:       meta.Converged,
	}
	if meta.PredictorMean != nil {
		result.PredictorMean = *meta.PredictorMean
	}
	if meta.PredictorStd != nil {
		result.PredictorStd = *meta.PredictorStd
	}
	return result
}

// loadParamsFromLookup populates a UnivariateModelResult from a tensor lookup function.
// The prefix should be like "zero_predictor/0/" or "univariate/1_0/".
func loadParamsFromLookup(lookup func(string) []float64, prefix string, result *UnivariateModelResult) {
	if data := lookup(prefix + "beta_mean"); data != nil {
		result.BetaMean = data
	}
	if data := lookup(prefix + "intercept_mean"); len(data) > 0 {
		result.InterceptMean = &data[0]
	}
	if data := lookup(prefix + "cutpoints_mean"); data != nil {
		result.CutpointsMean = data
	}
}

// LoadFromYAML loads a MiceBayesianLoo model from YAML bytes where
// parameters (beta_mean, intercept_mean, cutpoints_mean) are embedded
// directly in the YAML alongside metadata. No HDF5 file is needed.
func LoadFromYAML(yamlData []byte) (*MiceBayesianLoo, error) {
	var cfg configYAML
	if err := yaml.Unmarshal(yamlData, &cfg); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	varTypes := make(map[int]VariableType, len(cfg.Data.VariableTypes))
	for idx, vt := range cfg.Data.VariableTypes {
		varTypes[idx] = VariableType(vt)
	}

	zeroPredictors := make(map[int]*UnivariateModelResult, len(cfg.ZeroPredictorMeta))
	for key, meta := range cfg.ZeroPredictorMeta {
		targetIdx, err := strconv.Atoi(key)
		if err != nil {
			return nil, fmt.Errorf("parsing zero_predictor key %q: %w", key, err)
		}
		result := metaToResult(meta)
		result.PredictorIdx = -1
		result.TargetIdx = targetIdx
		loadParamsFromYAML(meta, result)
		zeroPredictors[targetIdx] = result
	}

	univariateModels := make(map[[2]int]*UnivariateModelResult, len(cfg.UnivariateMeta))
	for _, meta := range cfg.UnivariateMeta {
		result := metaToResult(meta)
		predictorIdx := 0
		if meta.PredictorIdx != nil {
			predictorIdx = *meta.PredictorIdx
		}
		result.PredictorIdx = predictorIdx
		result.TargetIdx = meta.TargetIdx
		loadParamsFromYAML(meta, result)
		key := [2]int{meta.TargetIdx, predictorIdx}
		univariateModels[key] = result
	}

	return &MiceBayesianLoo{
		Version:          cfg.Version,
		VariableNames:    cfg.Data.VariableNames,
		VariableTypes:    varTypes,
		NObs:             cfg.Data.NObsTotal,
		PredictionGraph:  cfg.PredictionGraph,
		ZeroPredictors:   zeroPredictors,
		UnivariateModels: univariateModels,
		MixedWeights:     cfg.MixedWeights,
	}, nil
}

func loadParamsFromYAML(meta univariateModelResultYAML, result *UnivariateModelResult) {
	if len(meta.BetaMean) > 0 {
		result.BetaMean = meta.BetaMean
	}
	if len(meta.InterceptMean) > 0 {
		result.InterceptMean = &meta.InterceptMean[0]
	}
	if len(meta.CutpointsMean) > 0 {
		result.CutpointsMean = meta.CutpointsMean
	}
}

// VariableIndex returns the index of a variable by name, or -1 if not found.
func (m *MiceBayesianLoo) VariableIndex(name string) int {
	for i, n := range m.VariableNames {
		if strings.EqualFold(n, name) {
			return i
		}
	}
	return -1
}
