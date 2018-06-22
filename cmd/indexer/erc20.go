// Copyright 2018 The eth-indexer Authors
// This file is part of the eth-indexer library.
//
// The eth-indexer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The eth-indexer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the eth-indexer library. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"encoding/json"
	"strconv"

	"github.com/getamis/sirius/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	erc20tokenCfgFileName = "erc20"
	erc20tokenCfgFileType = "yaml"
	erc20tokenCfgFilePath = "./configs"
)

var (
	list      map[string]interface{}
	addresses []string
	blocks    []int64
)

var vp = viper.New()

// LoadTokensFromConfig is the function to return addresses and blocks from config file
func LoadTokensFromConfig() ([]string, []int64, error) {
	for _, v := range list {
		data, _ := json.Marshal(v)
		result := make(map[string]string)
		err := json.Unmarshal(data, &result)
		if err != nil {
			return nil, nil, err
		}

		addr := result["address"]
		addresses = append(addresses, addr)

		block, _ := strconv.ParseInt(result["block"], 10, 64)
		blocks = append(blocks, block)
	}

	return addresses, blocks, nil
}

func init() {
	cobra.OnInitialize(initErc20TokenConfig)
}

func initErc20TokenConfig() {
	vp.SetConfigType(erc20tokenCfgFileType)
	vp.SetConfigName(erc20tokenCfgFileName)
	vp.AddConfigPath(erc20tokenCfgFilePath)

	if err := vp.ReadInConfig(); err != nil {
		log.Crit("Can not load config file", "err", err)
	}

	list = vp.GetStringMap(erc20tokenCfgFileName)
}
