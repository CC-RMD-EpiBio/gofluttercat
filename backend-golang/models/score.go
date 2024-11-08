package models

import (
	"fmt"

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
	AbilityGridPts     []float64
	GaussianPriorWidth float64
	Energy             []float64
	Model              IrtModel
}

type BayesianScore struct {
}

func (bs BayesianScorer) Score(resp *SessionResponses) error {
	abilities, err := ndvek.NewNdArray([]int{len(bs.AbilityGridPts)}, bs.AbilityGridPts)
	if err != nil {
		panic(err)
	}
	probs := bs.Model.Prob(abilities)

	fmt.Printf("probs: %v\n", probs)
	return nil
}

func (bs BayesianScorer) AddResponse() error {
	return nil
}

func NewBayesianScorer(AbilityGridPts []float64, GaussianPriorWidth float64, model IrtModel) *BayesianScorer {
	bs := &BayesianScorer{
		AbilityGridPts:     AbilityGridPts,
		GaussianPriorWidth: GaussianPriorWidth,
		Model:              model,
	}
	return bs
}
