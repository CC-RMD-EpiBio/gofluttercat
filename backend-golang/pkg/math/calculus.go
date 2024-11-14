package math

func Trapz(y []float64, dx float64) float64 {
	n := len(y)
	if n < 2 {
		return 0.0
	}

	integral := 0.0
	for i := 1; i < n; i++ {
		integral += (y[i] + y[i-1]) * dx / 2
	}

	return integral
}

func Trapz2(y, x []float64) float64 {
	if len(y) != len(x) {
		panic("y and x must have the same length")
	}

	n := len(y)
	if n < 2 {
		return 0.0
	}

	integral := 0.0
	for i := 1; i < n; i++ {
		dx := x[i] - x[i-1]
		integral += (y[i] + y[i-1]) * dx / 2
	}

	return integral
}
