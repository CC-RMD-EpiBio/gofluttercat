package math

import "math"

func Sigmoid(x float64) float64 {
	exp := math.Exp(x)
	return exp / (1 + exp)
}
