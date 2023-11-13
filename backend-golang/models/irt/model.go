package irt

type IRTModel interface {
	logLikelihood() float64
	fisherInformation() float64
}
