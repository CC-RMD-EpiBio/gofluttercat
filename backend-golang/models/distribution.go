package models

import "gorgonia.org/tensor"

type ContinuousDistribution interface {
	logProbs() tensor.Tensor
	probs() tensor.Tensor
}

type DiscreteDistribution interface {
	logProbs() tensor.Tensor
	probs() tensor.Tensor
}
