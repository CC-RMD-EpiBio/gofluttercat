package models

type CatSession struct {
	ModelBase
	Respondent Respondent
	Responses  []Response
}

type Response struct {
	Name  string
	Value int
	Item  *Item
}

type SessionResponses struct {
	Responses []Response
}

type SessionSavedState struct {
	Energy []float64
}
