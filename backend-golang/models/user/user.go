package user

import "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models"

type User struct {
	*models.ModelBase
	Email string
	Name  string
}
