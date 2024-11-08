package irt

import (
	"fmt"
	"math"
	"testing"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models"
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
				Discrimination: 1.0,
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
		[]*models.Item{&item1, &item2},
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

	probs := grm.Prob(ndvek.Zeros([]int{3}))
	fmt.Printf("probs: %v\n", probs)
	ll := grm.LogLikelihood(ndvek.Zeros([]int{3}), &sresponses)
	fmt.Printf("ll: %v\n", ll)
	prior := func(x float64) float64 {
		out := math.Exp(-x * x / 2)
		return out
	}

	scorer := models.NewBayesianScorer(ndvek.Linspace(-6, 6, 200), prior, grm)
	scores := scorer.Score(&sresponses)

	fmt.Printf("scores: %v\n", scores)
}
