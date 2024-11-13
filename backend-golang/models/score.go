package models

import (
	"fmt"
	"math"
	"slices"

	math2 "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/math"

	"github.com/mederrata/ndvek"
	"github.com/viterin/vek"
)

type Score interface {
	Mean() *ndvek.NdArray
	Std() *ndvek.NdArray
}

type Scorer interface {
	Score(*IrtModel, *SessionResponses) error
	RetrieveScore() Score
}
type BayesianScore struct {
	Energy []float64
	Grid   []float64
}

type BayesianScorer struct {
	AbilityGridPts []float64
	Prior          func(float64) float64
	Model          IrtModel
	Answered       []*Response
	Scored         map[string]int
	Running        *BayesianScore
}

func (bs BayesianScorer) Score(resp *SessionResponses) error {
	toAdd := make([]Response, 0)
	toDelete := make([]string, 0)
	for _, r := range resp.Responses {
		past, ok := bs.Scored[r.Item.Name]
		if ok {
			if past == r.Value {
				continue
			} else {
				toDelete = append(toDelete, r.Item.Name)
			}
		}
		toAdd = append(toAdd, r)
	}
	err := bs.RemoveResponses(toDelete)
	if err != nil {
		panic(err)
	}
	err = bs.AddResponses(toAdd)
	if err != nil {
		panic(err)
	}

	return nil
}

func (bs *BayesianScorer) AddResponses(resp []Response) error {
	if len(resp) == 0 {
		return nil
	}
	abilities, err := ndvek.NewNdArray([]int{len(bs.AbilityGridPts)}, bs.AbilityGridPts)
	if err != nil {
		panic(err)
	}

	ll := bs.Model.LogLikelihood(abilities, resp)
	for _, r := range resp {
		bs.Answered = append(bs.Answered, &r)
	}
	bs.Running.Energy = vek.Add(bs.Running.Energy, ll.Data)
	fmt.Printf("ll: %v\n", ll)

	return nil
}

func (bs *BayesianScorer) RemoveResponses(itmNames []string) error {
	toDelete := make([]Response, 0)
	toDeleteNames := make([]string, 0)
	for _, r := range bs.Answered {
		if slices.Contains(itmNames, r.Item.Name) {
			toDelete = append(toDelete, *r)
			toDeleteNames = append(toDeleteNames, r.Item.Name)
		}
	}

	n := 0
	for _, r := range bs.Answered {
		if !slices.Contains(toDeleteNames, r.Item.Name) {
			bs.Answered[n] = r
			n++
		}
	}
	bs.Answered = bs.Answered[:n]

	abilities, err := ndvek.NewNdArray([]int{len(bs.AbilityGridPts)}, bs.AbilityGridPts)
	if err != nil {
		panic(err)
	}
	ll := bs.Model.LogLikelihood(abilities, toDelete)
	bs.Running.Energy = vek.Sub(bs.Running.Energy, ll.Data)

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
		Running: &BayesianScore{
			Grid:   AbilityGridPts,
			Energy: energy,
		},
	}

	return bs
}

func (bs BayesianScore) Density() []float64 {
	d := make([]float64, len(bs.Energy))
	offset := vek.Min(bs.Energy)
	for i := 0; i < len(bs.Energy); i++ {
		d[i] = math.Exp(bs.Energy[i] - offset)
	}
	Z := math2.Trapz2(d, bs.Grid)
	d = vek.DivNumber(d, Z)
	return d
}

func (bs BayesianScore) Mean() float64 {
	d := bs.Density()
	mean := math2.Trapz2(vek.Mul(d, bs.Grid), bs.Grid)
	return mean
}

func (bs BayesianScore) Std() float64 {
	d := bs.Density()
	mean := math2.Trapz2(vek.Mul(d, bs.Grid), bs.Grid)
	second := math2.Trapz2(vek.Mul(bs.Grid, vek.Mul(d, bs.Grid)), bs.Grid)
	return second - mean*mean
}
