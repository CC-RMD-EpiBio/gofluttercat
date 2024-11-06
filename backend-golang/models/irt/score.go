package irt

import "github.com/mederrata/ndvek"

type Score struct {
}

type Scorer interface {
	Score(SessionResponses)
	GetScores() map[string]float64
}

type BayesianScorer struct {
	AbilityGridPts     ndvek.NdArray
	GaussianPriorWidth float64
}
