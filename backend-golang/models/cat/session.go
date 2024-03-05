package cat

import "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models"

type CatSession struct {
	models.ModelBase
	Respondent Respondent
	Responses  []Response
}
