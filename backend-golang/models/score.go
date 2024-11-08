package models

import (
	"fmt"
	"math"

	"github.com/mederrata/ndvek"
)

type Score interface {
	Mean() *ndvek.NdArray
	Std() *ndvek.NdArray
}

type Scorer interface {
	Score(*IrtModel, *SessionResponses) error
	RetrieveScore() Score
}

type BayesianScorer struct {
	AbilityGridPts []float64
	Prior          func(float64) float64
	Energy         []float64
	Model          IrtModel
	Answered       []*Item
}

type BayesianScore struct {
}

func (bs BayesianScorer) Score(resp *SessionResponses) error {
	abilities, err := ndvek.NewNdArray([]int{len(bs.AbilityGridPts)}, bs.AbilityGridPts)
	if err != nil {
		panic(err)
	}
	ll := bs.Model.LogLikelihood(abilities, resp)
	fmt.Printf("ll: %v\n", ll)
	return nil
}

func (bs BayesianScorer) AddResponse() error {
	return nil
}

func NewBayesianScorer(AbilityGridPts []float64, abilityPrior func(float64) float64, model IrtModel) *BayesianScorer {

	// initialize the prior
	energy := make([]float64, 0)
	for _, x := range AbilityGridPts {
		density := abilityPrior(x)
		energy = append(energy, math.Log(density))
	}

	bs := &BayesianScorer{
		AbilityGridPts: AbilityGridPts,
		Prior:          abilityPrior,
		Model:          model,
		Energy:         energy,
	}

	return bs
}
