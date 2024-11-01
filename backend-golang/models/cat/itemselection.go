package cat

import "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models/irt"

type ItemSelector interface {
	NextItem(*CatSession) *irt.Item
	ItemHistory() []*irt.Item
}
