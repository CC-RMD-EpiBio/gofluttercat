package rwas

import "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models"

type RwaSession struct {
	maxItems   int
	items      []*models.Item
	responses  []*models.Response
	respondent models.Respondent
}

func NewSession(maxItems int, respondent models.Respondent) *RwaSession {
	items := LoadItems("../../rwas/factorized")
	rwa := RwaSession{
		maxItems:   maxItems,
		items:      items,
		responses:  make([]*models.Response, 0),
		respondent: respondent,
	}
	return &rwa
}
