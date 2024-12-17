package cat

import "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models"

func ItemInList(items []*models.Item, item *models.Item) bool {
	for _, i := range items {
		if i.Name == item.Name {
			return true
		}
	}
	return false
}

func AdmissibleItems(bs *models.BayesianScorer) []*models.Item {
	answered := make([]*models.Item, 0)
	for _, i := range bs.Answered {
		answered = append(answered, i.Item)
	}
	admissible := make([]*models.Item, 0)
	allItems := bs.Model.GetItems()
	for _, it := range allItems {
		if !ItemInList(answered, it) {
			admissible = append(admissible, it)
		}
	}

	return admissible
}

func HasResponse(itemName string, responses []*models.Response) bool {
	for _, r := range responses {
		if r.Item.Name == itemName {
			return true
		}
	}
	return false
}
