package irt

import (
	"fmt"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models/irt"
	ndvek "github.com/mederrata/ndvek"
)

type GradedResponseModel struct {
	Scales          []*irt.Scale
	Items           []*irt.Item
	Discriminations ndvek.NdArray
	Difficulties    ndvek.NdArray
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

func (grm GradedResponseModel) LogLikelihood(abilities *ndvek.NdArray, resp *irt.SessionResponses) *ndvek.NdArray {
	// Shape of abilities is n_abilities x n_scale

	return nil
}

func (grm GradedResponseModel) Prob(abilities *ndvek.NdArray) *ndvek.NdArray {

	return nil
}
