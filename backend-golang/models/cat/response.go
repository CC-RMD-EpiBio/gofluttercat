package cat

import (
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models/irt"
)

type Response struct {
	models.ModelBase
	Item  *irt.Item
	Value string
}
