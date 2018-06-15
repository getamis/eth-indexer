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
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/getamis/eth-indexer/cmd/flags"
	"github.com/getamis/eth-indexer/service/erc20"
	"github.com/getamis/eth-indexer/service/indexer"
	"github.com/getamis/eth-indexer/store"
	"github.com/getamis/sirius/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	cfgFileName string = "config"
	cfgFileType string = "yaml"
	cfgFilePath string = "./configs"
)

var (
	configFile string

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

	// flags for syncing
	targetBlock int64
	fromBlock   int64

	// flags for erc20
	erc20Addresses    []string
	erc20BlockNumbers []int64

	// flags for profiling
	profiling  bool
	profilAddr string
	profilPort int
)

// RootCmd represents the base command when called without any subcommands
var ServerCmd = &cobra.Command{
	Use:   "indexer",
	Short: "blockchain data indexer",
	Long:  `blockchain data indexer`,
	RunE: func(cmd *cobra.Command, args []string) error {
		loadFlagToVar()

		// eth-client
		ethClient, err := NewEthConn(fmt.Sprintf("%s://%s:%d", ethProtocol, ethHost, ethPort))
		if err != nil {
			log.Error("Failed to new a eth client", "err", err)
			return err
		}
		defer ethClient.Close()

		// database
		db, err := NewDatabase()
		if err != nil {
			log.Error("Failed to connect to db", "err", err)
			return err
		}
		defer db.Close()

		sigs := make(chan os.Signal, 1)
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
			defer signal.Stop(sigs)

			log.Debug("Shutting down", "signal", <-sigs)
			cancel()
		}()

		indexer := indexer.New(ethClient, store.NewManager(db))
		erc20Addresses, erc20BlockNumbers, err := erc20.LoadTokenFromConfig()
		if err != nil {
			return err
		}
		if err := indexer.Init(ctx, erc20Addresses, erc20BlockNumbers); err != nil {
			return err
		}

		if profiling {
			// run `go tool pprof build/bin/service http://127.0.0.1:8000/debug/pprof/profile\?seconds\=60`
			// Start profiling
			go func() {
				url := fmt.Sprintf("%s:%d", profilAddr, profilPort)
				log.Info("Starting profiling", "url", url)
				http.ListenAndServe(url, nil)
			}()
		}

		if targetBlock > 0 {
			err = indexer.SyncToTarget(ctx, fromBlock, targetBlock)
		} else {
			ch := make(chan *types.Header)
			err = indexer.Listen(ctx, ch)
		}

		// Ignore if listener is stopped by signal
		if err == context.Canceled {
			return nil
		}
		cancel()
		return err
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
	root, _ := os.Getwd()
	cfgFile := root + "/" + cfgFilePath + "/" + cfgFileName + "." + cfgFileType

	// Take cfgFile as the first priority to load and only enable flags when cfgFile does not exists.
	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		fmt.Println("The config file does not exist. Run ServerCmd.Flags().")

		// eth-client flags
		ServerCmd.Flags().StringVar(&ethProtocol, flags.EthProtocolFlag, "ws", "The eth-client protocol")
		ServerCmd.Flags().StringVar(&ethHost, flags.EthHostFlag, "127.0.0.1", "The eth-client host")
		ServerCmd.Flags().IntVar(&ethPort, flags.EthPortFlag, 8546, "The eth-client port")

		// Database flags
		ServerCmd.Flags().StringVar(&dbDriver, flags.DbDriverFlag, "mysql", "The database driver")
		ServerCmd.Flags().StringVar(&dbHost, flags.DbHostFlag, "", "The database host")
		ServerCmd.Flags().IntVar(&dbPort, flags.DbPortFlag, 3306, "The database port")
		ServerCmd.Flags().StringVar(&dbName, flags.DbNameFlag, "ethdb", "The database name")
		ServerCmd.Flags().StringVar(&dbUser, flags.DbUserFlag, "root", "The database username to login")
		ServerCmd.Flags().StringVar(&dbPassword, flags.DbPasswordFlag, "my-secret-pw", "The database password to login")

		// Syncing related flags
		ServerCmd.Flags().Int64Var(&targetBlock, flags.SyncTargetBlockFlag, 0, "The block number to sync to initially")
		ServerCmd.Flags().Int64Var(&fromBlock, flags.SyncFromBlockFlag, 0, "The init block number to sync to initially")
		//ServerCmd.Flags().StringArrayVar(&erc20Addresses, "erc20.addresses", []string{}, "The addresses of erc20 token contracts")
		//ServerCmd.Flags().IntSliceVar(&erc20BlockNumber, "erc20.block-numbers", []int{}, "The block numbers as the erc20 contract is deployed")

		// Profling flags
		ServerCmd.Flags().BoolVar(&profiling, "pprof", false, "Enable the pprof HTTP server")
		ServerCmd.Flags().IntVar(&profilPort, "pprof.port", 8000, "pprof HTTP server listening port")
		ServerCmd.Flags().StringVar(&profilAddr, "pprof.address", "0.0.0.0", "pprof HTTP server listening interface")
	} else {
		cobra.OnInitialize(initConfig)
	}
}

func initConfig() {
	viper.SetConfigType(cfgFileType)
	viper.SetConfigName(cfgFileName)
	viper.AddConfigPath(cfgFilePath)
	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Can't read config:", err)
		os.Exit(1)
	}
}

func loadFlagToVar() {
	// flags for ethereum service
	ethProtocol = viper.GetString(flags.EthProtocolFlag)
	ethHost = viper.GetString(flags.EthHostFlag)
	ethPort = viper.GetInt(flags.EthPortFlag)

	// flags for database
	dbDriver = viper.GetString(flags.DbDriverFlag)
	dbHost = viper.GetString(flags.DbHostFlag)
	dbPort = viper.GetInt(flags.DbPortFlag)
	dbName = viper.GetString(flags.DbNameFlag)
	dbUser = viper.GetString(flags.DbUserFlag)
	dbPassword = viper.GetString(flags.DbPasswordFlag)

	// flags for syncing
	targetBlock = viper.GetInt64(flags.SyncTargetBlockFlag)

	//flag for pprof
	profiling = viper.GetBool(flags.PprofEnable)
	profilPort = viper.GetInt(flags.PprofPort)
	profilAddr = viper.GetString(flags.PprofAddress)
}
