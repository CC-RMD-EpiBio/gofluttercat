package irt

import "gorgonia.org/tensor"

type IRTModel interface {
	logLikelihood() float64
	fisherInformation() float64
	logProb(i Item, r Respondent) tensor.Tensor
}
