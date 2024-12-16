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

import (
	"math"
)

type CategoricalDistribution struct {
	Choices []string
	Probs   []float64
}
type UnivariateRealDistribution interface {
	Density(x float64) float64
	Mean()
	Variance()
	LogDensity(x float64) float64
}

func (c CategoricalDistribution) Sample() string {
	x := SampleCategorical(c.Probs)
	return c.Choices[x]
}

/*
// Golang doesn't have default methods
func (u UnivariateRealDistribution) LogDensity(x float64) float64 {
	return math.Log(u.Density(x))
}
*/

type GaussianDistribution struct {
	mu    float64
	sigma float64
}

func NewGaussianDistribution(mu float64, sigma float64) GaussianDistribution {
	return GaussianDistribution{mu: mu, sigma: sigma}
}

func (g GaussianDistribution) Mean() float64 {
	return g.mu
}

func (g GaussianDistribution) Variance() float64 {
	return math.Pow(g.sigma, 2)
}

func (g GaussianDistribution) Density(x float64) float64 {
	return math.Exp(-math.Pow((x-g.mu)/g.sigma, 2)/2) / math.Sqrt(2*math.Pi) / g.sigma
}

func (g GaussianDistribution) LogDensity(x float64) float64 {
	return -math.Pow((x-g.mu)/g.sigma, 2)/2 - 0.5*(math.Log(2*math.Pi)+2*math.Log(g.sigma))
}
