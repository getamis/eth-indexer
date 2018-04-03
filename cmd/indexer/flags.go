// Copyright © 2018 AMIS Technologies
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"strings"

	"github.com/spf13/viper"
)

const (
	configFileFlag = "config"

	// flags names for indexer
	startFlag  = "start"
	endFlag    = "end"
	listenFlag = "listen"

	// flag names for ethereum service
	ethProtocolFlag = "eth.protocol"
	ethHostFlag     = "eth.host"
	ethPortFlag     = "eth.port"

	// flag names for database
	dbDriverFlag   = "db.driver"
	dbHostFlag     = "db.host"
	dbPortFlag     = "db.port"
	dbNameFlag     = "db.name"
	dbUserFlag     = "db.user"
	dbPasswordFlag = "db.password"
)

var (
	configFile string

	// flags for indexer
	start  int64
	end    int64
	listen bool

	// flags for ethereum service
	ethProtocol string
	ethHost     string
	ethPort     int

	// flags for database
	dbDriver   string
	dbHost     string
	dbPort     int
	dbName     string
	dbUser     string
	dbPassword string
)

func loadConfigUsingViper(vp *viper.Viper, filename string) error {
	f, err := os.Open(strings.TrimSpace(filename))
	if err != nil {
		return err
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	vp.SetConfigType("yaml")
	vp.ReadConfig(bytes.NewBuffer(b))

	return nil
}

func loadFlagToVar(vp *viper.Viper) {
	// flags for ethereum service
	ethProtocol = vp.GetString(ethProtocolFlag)
	ethHost = vp.GetString(ethHostFlag)
	ethPort = vp.GetInt(ethPortFlag)

	// flags for database
	dbDriver = vp.GetString(dbDriverFlag)
	dbHost = vp.GetString(dbHostFlag)
	dbPort = vp.GetInt(dbPortFlag)
	dbName = vp.GetString(dbNameFlag)
	dbUser = vp.GetString(dbUserFlag)
	dbPassword = vp.GetString(dbPasswordFlag)
}
