package models

type ItemSelector interface {
	NextItem(*BayesianScorer) *Item
	Criterion(*BayesianScorer) map[string]float64
}
