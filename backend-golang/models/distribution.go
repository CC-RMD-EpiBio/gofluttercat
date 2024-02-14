package models

import "math"

type CategoricalDistribution struct {
}
type UnivariateRealDistribution interface {
	Density(x float64) float64
	Mean()
	Variance()
	LogDensity(x float64) float64
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
