package math

import (
	"math"
	"math/rand/v2"

	"github.com/viterin/vek"
)

func SampleCategorical(p []float64) int {
	r := rand.Float64()
	var cum float64 = 0
	var choice int = 0
	for _, value := range p {
		cum += value
		if r < cum {
			return choice
		}
		choice += 1
	}
	return choice
}

func EnergyToDensity(energy []float64, x []float64) []float64 {
	energy = vek.SubNumber(energy, vek.Min(energy))
	density := make([]float64, len(x))
	for i, e := range energy {
		density[i] = math.Exp(e)
	}
	Z := Trapz2(density, x)
	for i := range len(density) {
		density[i] /= Z
	}
	return density
}
