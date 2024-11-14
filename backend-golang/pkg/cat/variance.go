package cat

import (
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models"
)

type VarianceSelector struct {
	Temperature float64
}

func (vs VarianceSelector) NextItem(s *models.BayesianScorer) *models.Item {
	return nil
}
