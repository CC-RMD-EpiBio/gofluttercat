package cat

import (
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models/cat"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models/irt"
)

type KLSelector struct {
	Temperature float64
}

type FisherSelector struct {
	Temperature float64
}

type VarianceSelector struct {
	Temperature float64
}

func (KLSelector) NextItem(*cat.CatSession) *irt.Item {
	return nil
}

func (VarianceSelector) NextItem(*cat.CatSession) *irt.Item {
	return nil
}

func (FisherSelector) NextItem(*cat.CatSession) *irt.Item {
	return nil
}
