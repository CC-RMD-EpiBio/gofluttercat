package irt

import (
	"fmt"
	"math"
	"testing"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/cat"
	"github.com/mederrata/ndvek"
)

func Test_grm(t *testing.T) {
	item1 := models.Item{
		Name:     "Item1",
		Question: "I can walk",
		Choices: map[string]models.Choice{
			"Never": models.Choice{
				Text: "Never", Value: 1,
			},
			"Sometimes": models.Choice{
				Text:  "Sometimes",
				Value: 2},
			"Usually":       models.Choice{Text: "Usually", Value: 3},
			"Almost Always": models.Choice{Text: "Almost always", Value: 4},
			"Always":        models.Choice{Text: "Always", Value: 5}},
		ScaleLoadings: map[string]models.Calibration{
			"default": models.Calibration{
				Difficulties:   []float64{-2, -1, 1, 2},
				Discrimination: 1.0,
			},
		},
		ScoredValues: []int{1, 2, 3, 4, 5},
	}
	item2 := models.Item{
		Name:     "Item2",
		Question: "I can run",
		Choices: map[string]models.Choice{
			"Never": models.Choice{
				Text: "Never", Value: 1,
			},
			"Sometimes": models.Choice{
				Text:  "Sometimes",
				Value: 2},
			"Usually":       models.Choice{Text: "Usually", Value: 3},
			"Almost Always": models.Choice{Text: "Almost always", Value: 4},
			"Always":        models.Choice{Text: "Always", Value: 5}},
		ScaleLoadings: map[string]models.Calibration{
			"default": models.Calibration{
				Difficulties:   []float64{-1, -0.5, 1.5, 2.5},
				Discrimination: 2.0,
			},
		},
		ScoredValues: []int{1, 2, 3, 4, 5},
	}
	item3 := models.Item{
		Name:     "Item3",
		Question: "I can jump",
		Choices: map[string]models.Choice{
			"Never": models.Choice{
				Text: "Never", Value: 1,
			},
			"Sometimes": models.Choice{
				Text:  "Sometimes",
				Value: 2},
			"Usually":       models.Choice{Text: "Usually", Value: 3},
			"Almost Always": models.Choice{Text: "Almost always", Value: 4},
			"Always":        models.Choice{Text: "Always", Value: 5}},
		ScaleLoadings: map[string]models.Calibration{
			"default": models.Calibration{
				Difficulties:   []float64{0, 1, 2.5, 3.5},
				Discrimination: 3,
			},
		},
		ScoredValues: []int{1, 2, 3, 4, 5},
	}
	scale := models.Scale{
		Loc:   0,
		Scale: 1,
		Name:  "default",
	}
	grm := NewGRM(
		[]*models.Item{&item1, &item2, &item3},
		scale,
	)

	resp := models.Response{
		Name:  "Item1",
		Value: 1,
		Item:  &item1,
	}

	sresponses := models.SessionResponses{
		Responses: []models.Response{resp},
	}

	fmt.Printf("grm: %v\n", grm)

	abilities, err := ndvek.NewNdArray([]int{4}, []float64{0, -1, 1, 2})
	if err != nil {
		panic(err)
	}
	probs := grm.Prob(abilities)
	fmt.Printf("probs: %v\n", probs)
	ll := grm.LogLikelihood(abilities, sresponses.Responses)
	fmt.Printf("ll: %v\n", ll)
	prior := func(x float64) float64 {
		out := math.Exp(-x * x / 2)
		return out
	}

	scorer := models.NewBayesianScorer(ndvek.Linspace(-6, 6, 200), prior, grm)
	_ = scorer.Score(&sresponses)
	fmt.Printf("scorer.Running: %v\n", scorer.Running.Mean())

	selector := cat.FisherSelector{Temperature: 0}
	item := selector.NextItem(scorer)
	fmt.Printf("item: %v\n", item)

	bselector := cat.BayesianFisherSelector{Temperature: 0}
	bitem := bselector.NextItem(scorer)
	fmt.Printf("bitem: %v\n", bitem)

	kselector := cat.KLSelector{Temperature: 0}
	kitem := kselector.NextItem(scorer)
	fmt.Printf("kitem: %v\n", kitem)

	mckselector := cat.NewMcKlSelector(0, 200)
	mckitem := mckselector.NextItem(scorer)
	fmt.Printf("kitem: %v\n", mckitem)
}
