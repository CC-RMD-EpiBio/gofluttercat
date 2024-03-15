package irt

import (
	"fmt"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models/irt"
	ndvek "github.com/mederrata/ndvek"
)

type GradedResponseModel struct {
	Scales          []*irt.Scale
	Items           []*irt.Item
	Discriminations ndvek.NDArray
	Difficulties    ndvek.NDArray
}

func NewGRM(items []*irt.Item, scales []*irt.Scale) GradedResponseModel {
	model := GradedResponseModel{
		Items:  items,
		Scales: scales,
	}
	nItems := len(items)
	nScales := len(scales)
	discrimination := ndvek.Zeros([]int{nItems, nScales})
	fmt.Printf("discrimination: %v\n", discrimination)
	for _, item := range items {
		fmt.Printf("item: %v\n", item)
	}
	return model
}

func (grm GradedResponseModel) logLikelihood(abilities *ndvek.NDArray) ndvek.NDArray {
	// Shape of abilities is n_abilities x n_scale
	delta := grm.Difficulties.BroadcastOp(abilities, ndvek.Minus64)
	delta = delta.BroadcastOp(&grm.Discriminations, ndvek.Mul64)
	fmt.Printf("delta: %v\n", delta)
	return *delta
}

func (grm GradedResponseModel) probs(abilities *ndvek.NDArray) ndvek.NDArray {
	delta := grm.Difficulties.BroadcastOp(abilities, ndvek.Minus64)
	delta = delta.BroadcastOp(&grm.Discriminations, ndvek.Mul64)
	return *delta
}
