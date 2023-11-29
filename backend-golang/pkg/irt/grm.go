package irt

import (
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models/irt"
	"gorgonia.org/tensor"
)

type GradedResponseModel struct {
	Scales []string
	Items  []*irt.Item
}

func (grm GradedResponseModel) logLikelihood(ability tensor.Tensor) {
	for i, item := range grm.Items {
		print(i)
		print(item)
	}
}
