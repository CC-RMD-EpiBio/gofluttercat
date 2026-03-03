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
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
)

// safetensorsMeta describes one tensor entry in the safetensors header.
type safetensorsMeta struct {
	Dtype       string    `json:"dtype"`
	Shape       []int     `json:"shape"`
	DataOffsets [2]uint64 `json:"data_offsets"`
}

// readSafetensors reads a safetensors file and returns a map of tensor name
// to float64 slices. Only F64 and F32 dtypes are supported; F32 values are
// promoted to float64.
func readSafetensors(path string) (map[string][]float64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading safetensors file: %w", err)
	}

	if len(data) < 8 {
		return nil, fmt.Errorf("safetensors file too short")
	}

	headerSize := binary.LittleEndian.Uint64(data[:8])
	headerEnd := 8 + headerSize
	if uint64(len(data)) < headerEnd {
		return nil, fmt.Errorf("safetensors file truncated: header says %d bytes but file has %d", headerEnd, len(data))
	}

	// Parse JSON header. The header is a map of tensor-name -> metadata,
	// plus an optional "__metadata__" key with string-string metadata.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data[8:headerEnd], &raw); err != nil {
		return nil, fmt.Errorf("parsing safetensors header: %w", err)
	}

	tensorData := data[headerEnd:]
	result := make(map[string][]float64, len(raw))

	for name, rawVal := range raw {
		if name == "__metadata__" {
			continue
		}

		var meta safetensorsMeta
		if err := json.Unmarshal(rawVal, &meta); err != nil {
			return nil, fmt.Errorf("parsing tensor %q metadata: %w", name, err)
		}

		start := meta.DataOffsets[0]
		end := meta.DataOffsets[1]
		if end > uint64(len(tensorData)) || start > end {
			return nil, fmt.Errorf("tensor %q: invalid data offsets [%d, %d)", name, start, end)
		}
		blob := tensorData[start:end]

		switch meta.Dtype {
		case "F64":
			n := len(blob) / 8
			vals := make([]float64, n)
			for i := range n {
				vals[i] = math.Float64frombits(binary.LittleEndian.Uint64(blob[i*8 : (i+1)*8]))
			}
			result[name] = vals

		case "F32":
			n := len(blob) / 4
			vals := make([]float64, n)
			for i := range n {
				bits := binary.LittleEndian.Uint32(blob[i*4 : (i+1)*4])
				vals[i] = float64(math.Float32frombits(bits))
			}
			result[name] = vals

		default:
			return nil, fmt.Errorf("tensor %q: unsupported dtype %q (only F64 and F32 are supported)", name, meta.Dtype)
		}
	}

	return result, nil
}
