package rwas

import (
	"fmt"
	"math"
	"time"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models"
)


type RwaSession struct {
	maxItems   int
	items      []*models.Item
	responses  []*models.Response
	respondent *models.Respondent
	Scorer     map[string]models.Scorer
	StartTime  *time.Time
}

func NewSession(maxItems int, respondent *models.Respondent) *RwaSession {
	items := LoadItems()
	prior := func(x float64) float64 {
		out := math.Exp(-x * x / 2)
		return out
	}
	test := prior(0)
	fmt.Printf("test: %v\n", test)
	// models.NewBayesianScorer(ndvek.Linspace(-10, 10, 400), prior)
	rwa := RwaSession{
		maxItems:   maxItems,
		items:      items,
		responses:  make([]*models.Response, 0),
		respondent: respondent,
	}
	return &rwa
}
