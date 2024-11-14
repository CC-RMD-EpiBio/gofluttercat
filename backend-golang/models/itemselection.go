package models

type ItemSelector interface {
	NextItem(*CatSession) *Item
	ItemHistory() []*Item
}
