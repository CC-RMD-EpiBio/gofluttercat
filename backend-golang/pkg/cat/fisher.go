package cat

import (
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models"
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

func (ks KLSelector) NextItem(*models.CatSession) *models.Item {
	return nil
}

func (vs VarianceSelector) NextItem(*models.CatSession) *models.Item {
	return nil
}

func (fs FisherSelector) NextItem(*models.CatSession) *models.Item {
	return nil
}
