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
	Version string `yaml:"version"`
	Data    struct {
		VariableTypes map[int]string `yaml:"variable_types"`
		VariableNames []string       `yaml:"variable_names"`
		NObsTotal     int            `yaml:"n_obs_total"`
	} `yaml:"data"`
	PredictionGraph   map[string][]string                  `yaml:"prediction_graph"`
	ZeroPredictorMeta map[string]univariateModelResultYAML `yaml:"zero_predictor_meta"`
	UnivariateMeta    []univariateModelResultYAML          `yaml:"univariate_meta"`
}

type univariateModelResultYAML struct {
	PredictorIdx    *int     `yaml:"predictor_idx"`
	PredictorMean   *float64 `yaml:"predictor_mean"`
	PredictorStd    *float64 `yaml:"predictor_std"`
	NObs            int      `yaml:"n_obs"`
	ElpdLoo         float64  `yaml:"elpd_loo"`
	ElpdLooPerObs   float64  `yaml:"elpd_loo_per_obs"`
	ElpdLooPerObsSe float64  `yaml:"elpd_loo_per_obs_se"`
	KhatMax         float64  `yaml:"khat_max"`
	KhatMean        float64  `yaml:"khat_mean"`
	TargetIdx       int      `yaml:"target_idx"`
	Converged       bool     `yaml:"converged"`
}

// LoadFromDisk loads a MiceBayesianLoo model from a directory containing
// config.yaml and params.h5, as produced by bayesianquilts save_to_disk.
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

	// 2. Read params.h5
	h5Path := filepath.Join(dirPath, "params.h5")
	h5File, err := hdf5.Open(h5Path)
	if err != nil {
		return nil, fmt.Errorf("opening params.h5: %w", err)
	}
	defer h5File.Close()

	// Index all HDF5 datasets by path
	datasets := make(map[string]*hdf5.Dataset)
	h5File.Walk(func(path string, obj hdf5.Object) {
		if ds, ok := obj.(*hdf5.Dataset); ok {
			datasets[path] = ds
		}
	})

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

		prefix := fmt.Sprintf("/zero_predictor/%d/", targetIdx)
		if err := loadParams(datasets, prefix, result); err != nil {
			return nil, fmt.Errorf("loading zero_predictor/%d params: %w", targetIdx, err)
		}

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

		prefix := fmt.Sprintf("/univariate/%d_%d/", meta.TargetIdx, predictorIdx)
		if err := loadParams(datasets, prefix, result); err != nil {
			return nil, fmt.Errorf("loading univariate/%d_%d params: %w", meta.TargetIdx, predictorIdx, err)
		}

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

func loadParams(datasets map[string]*hdf5.Dataset, prefix string, result *UnivariateModelResult) error {
	if ds, ok := datasets[prefix+"beta_mean"]; ok {
		data, err := ds.Read()
		if err != nil {
			return fmt.Errorf("reading beta_mean: %w", err)
		}
		result.BetaMean = data
	}

	if ds, ok := datasets[prefix+"intercept_mean"]; ok {
		data, err := ds.Read()
		if err != nil {
			return fmt.Errorf("reading intercept_mean: %w", err)
		}
		if len(data) > 0 {
			result.InterceptMean = &data[0]
		}
	}

	if ds, ok := datasets[prefix+"cutpoints_mean"]; ok {
		data, err := ds.Read()
		if err != nil {
			return fmt.Errorf("reading cutpoints_mean: %w", err)
		}
		result.CutpointsMean = data
	}

	return nil
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
