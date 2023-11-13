package irt

type Scale struct {
	name  string
	loc   float64
	scale float64
}

func (s Scale) rescale([]float64) []float64 {
	return nil
}
