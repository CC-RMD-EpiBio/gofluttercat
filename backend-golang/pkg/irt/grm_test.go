package irt

import (
	"fmt"
	"testing"

	irtmodels "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models/irt"
)

func Test_grm(t *testing.T) {
	item1 := irtmodels.Item{
		Name:     "Item1",
		Question: "I can walk",
	}
	item2 := irtmodels.Item{
		Name:     "Item2",
		Question: "I can swim",
	}
	scale := irtmodels.Scale{
		Loc:   0,
		Scale: 1,
		Name:  "default",
	}
	grm := NewGRM(
		[]*irtmodels.Item{&item1, &item2},
		[]*irtmodels.Scale{&scale},
	)
	fmt.Printf("grm: %v\n", grm)
}
