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
