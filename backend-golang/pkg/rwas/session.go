package rwas

import "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models"

type RwaSession struct {
	maxItems   int
	items      []*models.Item
	responses  []*models.Response
	respondent models.Respondent
}

func NewSession() RwaSession {
	items := LoadItems()
}
