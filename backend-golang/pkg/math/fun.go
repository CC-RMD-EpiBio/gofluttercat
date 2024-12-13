package math

import "math"

func Sigmoid(x float64) float64 {
	exp := math.Exp(x)
	return exp / (1 + exp)
}

func Xlogy(x, y float64) float64 {
	if x == 0 {
		return 0
	}
	return x * math.Log(y)
}
