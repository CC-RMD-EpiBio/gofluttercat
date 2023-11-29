package irt

type IRTModel interface {
	logLikelihood() float64
	fisherInformation() float64
	Prob(i Item, r Respondent) float64
}

type GradedResponseModel struct {
}

type Ability struct {
	Scores map[Scale]float64
}

type Distribution interface {
	logProb(x interface{}) float64
}
