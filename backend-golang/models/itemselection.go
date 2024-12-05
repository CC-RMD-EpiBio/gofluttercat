package models

type ItemSelector interface {
	NextItem(*CatSession) *Item
	Criterion(*CatSession) map[string]float64
	ItemHistory() []*Item
}
