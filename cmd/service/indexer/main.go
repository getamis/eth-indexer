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

package indexer

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/getamis/sirius/log"
	"github.com/maichain/eth-indexer/cmd/flags"
	"github.com/maichain/eth-indexer/service/indexer"
	manager "github.com/maichain/eth-indexer/store/store_manager"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	configFile string

	// flags for indexer
	start int64
	end   int64

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

// RootCmd represents the base command when called without any subcommands
var ServerCmd = &cobra.Command{
	Use:   "indexer",
	Short: "blockchain data indexer",
	Long:  `blockchain data indexer`,
	Run: func(cmd *cobra.Command, args []string) {
		vp := viper.New()
		vp.BindPFlags(cmd.Flags())
		vp.AutomaticEnv() // read in environment variables that match
		if configFile := vp.GetString(flags.ConfigFileFlag); configFile != "" {
			if err := loadConfigUsingViper(vp, configFile); err != nil {
				log.Error("Failed to load config file", "err", err)
				return
			}
			loadFlagToVar(vp)
		}

		// eth-client
		ethClient := MustEthConn(fmt.Sprintf("%s://%s:%d", ethProtocol, ethHost, ethPort))
		// log.Info("eth=client" + ethClient)

		// database
		db := MustNewDatabase()
		defer db.Close()

		store := manager.NewStoreManager(db)
		indexer := indexer.NewIndexer(ethClient, store)
		indexer.Start(start, end)

		return
	},
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := ServerCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	// indexer flags
	ServerCmd.Flags().Int64Var(&start, flags.StartFlag, 0, "The start block height")
	ServerCmd.Flags().Int64Var(&end, flags.EndFlag, 0, "The end block height")

	// eth-client flags
	ServerCmd.Flags().StringVar(&ethProtocol, flags.EthProtocolFlag, "ws", "The eth-client protocol")
	ServerCmd.Flags().StringVar(&ethHost, flags.EthHostFlag, "127.0.0.1", "The eth-client host")
	ServerCmd.Flags().IntVar(&ethPort, flags.EthPortFlag, 8546, "The eth-client port")

	// Database flags
	ServerCmd.Flags().StringVar(&dbDriver, flags.DbDriverFlag, "mysql", "The database driver")
	ServerCmd.Flags().StringVar(&dbHost, flags.DbHostFlag, "", "The database host")
	ServerCmd.Flags().IntVar(&dbPort, flags.DbPortFlag, 3306, "The database port")
	ServerCmd.Flags().StringVar(&dbName, flags.DbNameFlag, "eth-db", "The database name")
	ServerCmd.Flags().StringVar(&dbUser, flags.DbUserFlag, "root", "The database username to login")
	ServerCmd.Flags().StringVar(&dbPassword, flags.DbPasswordFlag, "my-secret-pw", "The database password to login")
}

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
	ethProtocol = vp.GetString(flags.EthProtocolFlag)
	ethHost = vp.GetString(flags.EthHostFlag)
	ethPort = vp.GetInt(flags.EthPortFlag)

	// flags for database
	dbDriver = vp.GetString(flags.DbDriverFlag)
	dbHost = vp.GetString(flags.DbHostFlag)
	dbPort = vp.GetInt(flags.DbPortFlag)
	dbName = vp.GetString(flags.DbNameFlag)
	dbUser = vp.GetString(flags.DbUserFlag)
	dbPassword = vp.GetString(flags.DbPasswordFlag)
}
