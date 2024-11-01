package irt

import (
	"encoding/json"
	"log"
	"os"
)

type Item struct {
	Name          string                 `json:"item"`
	Question      string                 `json:"question"`
	Choices       map[string]Choice      `json:"responses"`
	ScaleLoadings map[string]Calibration `json:"scales"`
	Version       float32                `json:"version"`
}

type ItemDb struct {
	Items *[]Item
}

type Choice struct {
	Text  string `json:"text"`
	Value uint   `json:"value"`
}

type Calibration struct {
	Difficulties   []float64 `json:"difficulties"`
	Discrimination float64   `json:"discrimination"`
}

func LoadItem(path string) *Item {
	dat, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	var item *Item
	if err := json.Unmarshal(dat, &item); err != nil {
		log.Fatal(err)
	}
	return item
}
