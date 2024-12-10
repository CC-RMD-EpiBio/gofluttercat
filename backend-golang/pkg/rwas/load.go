package rwas

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"

	irtmodels "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/irt"
	rwasmodel "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/rwas"
)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func LoadScales(path string) map[string]*irtmodels.Scale {
	dat, err := os.ReadFile(path)
	check(err)
	var c map[string]*irtmodels.Scale
	if err := json.Unmarshal(dat, &c); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("rwasmodel.FactorizedDir: %v\n", rwasmodel.FactorizedDir)
	return c

}

func LoadItems() []*irtmodels.Item {
	cached, err := fs.ReadDir(rwasmodel.FactorizedDir, "factorized")
	check(err)

	var items []*irtmodels.Item
	for _, fn := range cached {
		d, err := fs.ReadFile(rwasmodel.FactorizedDir, "factorized/"+fn.Name())
		check(err)

		newItem := irtmodels.LoadItemS(d, []int{1, 2, 3, 4, 5, 6, 7, 8, 9})
		if newItem != nil {
			items = append(items, newItem)
		}

		fmt.Printf("fn: %v\n", fn)
	}

	return items
}

func LoadAutoencodedItems() []*irtmodels.Item {
	cached, err := fs.ReadDir(rwasmodel.AutoencodedDir, "autoencoded")
	check(err)

	var items []*irtmodels.Item
	for _, fn := range cached {
		d, err := fs.ReadFile(rwasmodel.AutoencodedDir, "autoencoded/"+fn.Name())
		check(err)

		newItem := irtmodels.LoadItemS(d, []int{1, 2, 3, 4, 5, 6, 7, 8, 9})
		if newItem != nil {
			items = append(items, newItem)
		}

		fmt.Printf("fn: %v\n", fn)
	}

	return items
}

func Load(path string) map[string]irt.GradedResponseModel {
	items := LoadItems()
	scales := LoadScales(path)
	models := make(map[string]irt.GradedResponseModel, 0)
	for scaleName, scale := range scales {
		it := make([]*irtmodels.Item, 0)
		for _, itm := range items {
			_, ok := itm.ScaleLoadings[scaleName]
			if ok {
				it = append(it, itm)
			}
		}
		models[scaleName] = irt.NewGRM(it, *scale)
	}
	return models
}
