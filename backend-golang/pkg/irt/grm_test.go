package irt

import (
	"fmt"
	"testing"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models/irt"
	"github.com/mederrata/ndvek"
)

func Test_grm(t *testing.T) {
	item1 := irt.Item{
		Name:     "Item1",
		Question: "I can walk",
		Choices: map[string]irt.Choice{
			"Never": irt.Choice{
				Text: "Never", Value: 1,
			},
			"Sometimes": irt.Choice{
				Text:  "Sometimes",
				Value: 2},
			"Usually":       irt.Choice{Text: "Usually", Value: 3},
			"Almost Always": irt.Choice{Text: "Almost always", Value: 4},
			"Always":        irt.Choice{Text: "Always", Value: 5}},
		ScaleLoadings: map[string]irt.Calibration{
			"default": irt.Calibration{
				Difficulties:   []float64{-2, -1, 1, 2},
				Discrimination: 1.0,
			},
		},
	}
	item2 := irt.Item{
		Name:     "Item2",
		Question: "I can run",
		Choices: map[string]irt.Choice{
			"Never": irt.Choice{
				Text: "Never", Value: 1,
			},
			"Sometimes": irt.Choice{
				Text:  "Sometimes",
				Value: 2},
			"Usually":       irt.Choice{Text: "Usually", Value: 3},
			"Almost Always": irt.Choice{Text: "Almost always", Value: 4},
			"Always":        irt.Choice{Text: "Always", Value: 5}},
		ScaleLoadings: map[string]irt.Calibration{
			"default": irt.Calibration{
				Difficulties:   []float64{-1, -0.5, 1.5, 2.5},
				Discrimination: 1.0,
			},
		},
	}
	scale := irt.Scale{
		Loc:   0,
		Scale: 1,
		Name:  "default",
	}
	grm := NewGRM(
		[]*irt.Item{&item1, &item2},
		scale,
	)

	fmt.Printf("grm: %v\n", grm)

	probs := grm.Prob(ndvek.Zeros([]int{1}))
	fmt.Printf("probs: %v\n", probs)
}
