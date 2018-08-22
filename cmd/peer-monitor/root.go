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
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getamis/sirius/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	ethURLFlag          = "eth.url"
	monitorDurationFlag = "monitor.duration"
	minPeerCountFlag    = "peercount.min"
	maxPeerCountFlag    = "peercount.max"
)

var (
	// flags for ethereum service
	ethURL string
	// flags for monitor
	monitorDuration time.Duration
	minPeerCount    int
	maxPeerCount    int
)

var ServerCmd = &cobra.Command{
	Use:   "peer-monitor",
	Short: "peer-monitor runs peer monitor",
	Long:  `peer-monitor runs peer monitor`,
	RunE: func(cmd *cobra.Command, args []string) error {
		peerMonitor := NewPeerMonitor(ethURL, minPeerCount, maxPeerCount)

		go func() {
			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
			defer signal.Stop(sigs)
			log.Info("Shutting down", "signal", <-sigs)
			peerMonitor.Stop()
		}()

		log.Info("Ready to monitor")
		err := peerMonitor.Run(monitorDuration)
		if err != nil {
			log.Error("Stopped unexpectedly", "err", err)
		}

		return err
	},
}

var onceCmd = &cobra.Command{
	Use:   "once",
	Short: "once runs peer monitor once",
	Long:  `once runs peer monitor once`,
	RunE: func(cmd *cobra.Command, args []string) error {
		peerMonitor := NewPeerMonitor(ethURL, minPeerCount, maxPeerCount)
		return peerMonitor.RunOnce()
	},
}

func Execute() {
	if err := ServerCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initViper)

	ServerCmd.AddCommand(onceCmd)

	// eth-client flags
	ServerCmd.PersistentFlags().String(ethURLFlag, "/user/.ethereum/geth.ipc", "The url of eth server")
	ServerCmd.PersistentFlags().Int(minPeerCountFlag, 5, "Minimum number of peer count")
	ServerCmd.PersistentFlags().Int(maxPeerCountFlag, 15, "Maximum number of peer count")

	ServerCmd.Flags().Duration(monitorDurationFlag, 1*time.Hour, "Monitor duration for eth peer set")

}

func initViper() {
	// Viper uses the following precedence order. Each item takes precedence over the item below it: 1st. flag. 2nd.config
	viper.BindPFlags(ServerCmd.Flags())

	// assign variables from Viper
	assignVarFromViper()
}

func assignVarFromViper() {
	// eth-client flags
	ethURL = viper.GetString(ethURLFlag)
	monitorDuration = viper.GetDuration(monitorDurationFlag)
	minPeerCount = viper.GetInt(minPeerCountFlag)
	maxPeerCount = viper.GetInt(maxPeerCountFlag)
}
