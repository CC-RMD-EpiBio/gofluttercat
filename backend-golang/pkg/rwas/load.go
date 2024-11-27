package rwas

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	irtmodels "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models"
	"github.com/yargevad/filepathx"
)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func LoadScales(path string) *map[string]*irtmodels.Scale {
	dat, err := os.ReadFile(path)
	check(err)
	var c *map[string]*irtmodels.Scale
	if err := json.Unmarshal(dat, &c); err != nil {
		log.Fatal(err)
	}
	return c

}

func LoadItems(path string) []*irtmodels.Item {
	g, err := filepathx.Glob(path + "**/*.json")
	check(err)
	var items []*irtmodels.Item
	for _, fn := range g {
		newItem := irtmodels.LoadItem(fn)
		if newItem != nil {
			items = append(items, newItem)
		}

		fmt.Printf("fn: %v\n", fn)
	}

	fmt.Printf("g: %v\n", g)
	return items
}
