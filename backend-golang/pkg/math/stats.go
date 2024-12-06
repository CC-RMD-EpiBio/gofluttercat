package math

import "math/rand/v2"

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
