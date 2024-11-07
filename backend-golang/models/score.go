package models

type Score struct {
}

type Scorer interface {
	Score(SessionResponses)
	GetScores() map[string]float64
}

type BayesianScorer struct {
	AbilityGridPts     []float64
	GaussianPriorWidth float64
	Energy             []float64
}
