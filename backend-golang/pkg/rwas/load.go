/*
###############################################################################
#
#                           COPYRIGHT NOTICE
#                  Mark O. Hatfield Clinical Research Center
#                       National Institutes of Health
#            United States Department of Health and Human Services
#
# This software was developed and is owned by the National Institutes of
# Health Clinical Center (NIHCC), an agency of the United States Department
# of Health and Human Services, which is making the software available to the
# public for any commercial or non-commercial purpose under the following
# open-source BSD license.
#
# Redistribution and use in source and binary forms, with or without
# modification, are permitted provided that the following conditions are met:
#
# (1) Redistributions of source code must retain this copyright
# notice, this list of conditions and the following disclaimer.
#
# (2) Redistributions in binary form must reproduce this copyright
# notice, this list of conditions and the following disclaimer in the
# documentation and/or other materials provided with the distribution.
#
# (3) Neither the names of the National Institutes of Health Clinical
# Center, the National Institutes of Health, the U.S. Department of
# Health and Human Services, nor the names of any of the software
# developers may be used to endorse or promote products derived from
# this software without specific prior written permission.
#
# (4) Please acknowledge NIHCC as the source of this software by including
# the phrase "Courtesy of the U.S. National Institutes of Health Clinical
# Center"or "Source: U.S. National Institutes of Health Clinical Center."
#
# THIS SOFTWARE IS PROVIDED BY THE U.S. GOVERNMENT AND CONTRIBUTORS "AS
# IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED
# TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A
# PARTICULAR PURPOSE ARE DISCLAIMED.
#
# You are under no obligation whatsoever to provide any bug fixes,
# patches, or upgrades to the features, functionality or performance of
# the source code ("Enhancements") to anyone; however, if you choose to
# make your Enhancements available either publicly, or directly to
# the National Institutes of Health Clinical Center, without imposing a
# separate written license agreement for such Enhancements, then you hereby
# grant the following license: a non-exclusive, royalty-free perpetual license
# to install, use, modify, prepare derivative works, incorporate into
# other computer software, distribute, and sublicense such Enhancements or
# derivative works thereof, in binary and source code form.
#
###############################################################################
*/

package rwas

import (
	"encoding/json"
	"io/fs"
	"log"
	"os"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/irtcat"
	rwasmodel "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/rwas"
)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func LoadScales(path string) map[string]*irtcat.Scale {
	dat, err := os.ReadFile(path)
	check(err)
	var c map[string]*irtcat.Scale
	if err := json.Unmarshal(dat, &c); err != nil {
		log.Fatal(err)
	}
	return c

}

func LoadItems() []*irtcat.Item {
	cached, err := fs.ReadDir(rwasmodel.FactorizedDir, "factorized")
	check(err)

	var items []*irtcat.Item
	for _, fn := range cached {
		d, err := fs.ReadFile(rwasmodel.FactorizedDir, "factorized/"+fn.Name())
		check(err)

		newItem := irtcat.LoadItemS(d, []int{1, 2, 3, 4, 5, 6, 7, 8, 9})
		if newItem != nil {
			items = append(items, newItem)
		}

	}

	return items
}

func LoadAutoencodedItems() []*irtcat.Item {
	cached, err := fs.ReadDir(rwasmodel.AutoencodedDir, "autoencoded")
	check(err)

	var items []*irtcat.Item
	for _, fn := range cached {
		d, err := fs.ReadFile(rwasmodel.AutoencodedDir, "autoencoded/"+fn.Name())
		check(err)

		newItem := irtcat.LoadItemS(d, []int{1, 2, 3, 4, 5, 6, 7, 8, 9})
		if newItem != nil {
			items = append(items, newItem)
		}

	}

	return items
}

func Load() map[string]*irtcat.GradedResponseModel {
	items := LoadItems()
	scales := make(map[string]*irtcat.Scale, 0)
	scales["A"] = &irtcat.Scale{
		Loc:     0,
		Scale:   1,
		Name:    "A",
		Version: 1.0,
	}
	scales["B"] = &irtcat.Scale{
		Loc:     0,
		Scale:   1,
		Name:    "B",
		Version: 1.0,
	}
	models := make(map[string]*irtcat.GradedResponseModel, 0)
	for scaleName, scale := range scales {
		it := make([]*irtcat.Item, 0)
		for _, itm := range items {
			_, ok := itm.ScaleLoadings[scaleName]
			if ok {
				it = append(it, itm)
			}
		}
		mod := irtcat.NewGRM(it, *scale)
		models[scaleName] = &mod
	}
	return models
}
