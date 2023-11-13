package cat

import "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models/irt"

type ItemSelector interface {
	nextItem() *irt.Item
	itemHistory() []*irt.Item
}
