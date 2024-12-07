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
	d := make([]float64, len(energy))
	offset := vek.Min(energy)
	for i := 0; i < len(energy); i++ {
		d[i] = math.Exp(energy[i] - offset)
	}
	Z := Trapz2(d, x)
	d = vek.DivNumber(d, Z)
	return d
}
