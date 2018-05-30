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
//

package rpc

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/getamis/sirius/log"
	"github.com/getamis/sirius/metrics"
	generalRPC "github.com/getamis/sirius/rpc"
	"github.com/maichain/eth-indexer/cmd/flags"
	"github.com/maichain/eth-indexer/service/rpc"
	"github.com/maichain/eth-indexer/store"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

var (
	host string
	port int

	// flags for ethereum service
	eth         bool
	ethProtocol string
	ethHost     string
	ethPort     int

	dbDriver   string
	dbHost     string
	dbPort     int
	dbName     string
	dbUser     string
	dbPassword string

	// flags for metrics
	metricsEnabled bool
	metricsHost    string
	metricsPort    int
)

// ServerCmd represents the grpc-server command
var ServerCmd = &cobra.Command{
	Use:   "rpc",
	Short: "rpc runs a gRPC server for api service",
	Long:  `rpc runs a gRPC server for api service.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
		if err != nil {
			log.Error("Failed to listen", "host", host, "port", port, "err", err)
			return err
		}

		var httpServer *http.Server
		if metricsEnabled {
			serveMux := http.NewServeMux()
			serveMux.HandleFunc("/metrics", metrics.DefaultRegistry.ServeHTTP)
			httpServer = &http.Server{
				Addr:    fmt.Sprintf("%s:%d", metricsHost, metricsPort),
				Handler: serveMux,
			}
			// Shutdown http server before termination
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				httpServer.Shutdown(ctx)
			}()
			go func() {
				if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
					log.Crit("Http Server stopped unexpectedly", "err", err)
				}
			}()
		}

		var s *generalRPC.Server
		if eth {
			// eth-client
			ethClient, err := NewEthConn(fmt.Sprintf("%s://%s:%d", ethProtocol, ethHost, ethPort))
			if err != nil {
				log.Error("Failed to new a eth client", "err", err)
				return err
			}
			defer ethClient.Close()

			s = generalRPC.NewServer(
				generalRPC.APIs(rpc.NewRelay(ethClient)),
				generalRPC.Metrics(metrics.NewServerMetrics(metrics.Namespace("indexerRelay"))),
			)
		} else {
			db := MustNewDatabase()
			defer db.Close()
			s = generalRPC.NewServer(
				generalRPC.APIs(rpc.New(store.NewServiceManager(db))),
				generalRPC.Metrics(metrics.NewServerMetrics(metrics.Namespace("indexerDB"))),
			)
		}

		if err := s.Serve(l); err != grpc.ErrServerStopped {
			log.Crit("Server stopped unexpectedly", "err", err)
		}

		return nil
	},
}

func init() {
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// ServerCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// grpc-ServerCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	ServerCmd.Flags().StringVar(&host, flags.Host, "", "The gRPC server listening host")
	ServerCmd.Flags().IntVar(&port, flags.Port, 5487, "The gRPC server listening port")

	// Database flags
	ServerCmd.Flags().StringVar(&dbDriver, flags.DbDriverFlag, "mysql", "The database driver")
	ServerCmd.Flags().StringVar(&dbHost, flags.DbHostFlag, "", "The database host")
	ServerCmd.Flags().IntVar(&dbPort, flags.DbPortFlag, 3306, "The database port")
	ServerCmd.Flags().StringVar(&dbName, flags.DbNameFlag, "eth-db", "The database name")
	ServerCmd.Flags().StringVar(&dbUser, flags.DbUserFlag, "root", "The database username to login")
	ServerCmd.Flags().StringVar(&dbPassword, flags.DbPasswordFlag, "my-secret-pw", "The database password to login")

	// eth-client flags
	ServerCmd.Flags().BoolVar(&eth, flags.EthFlag, false, "Enable Ethereum relay")
	ServerCmd.Flags().StringVar(&ethProtocol, flags.EthProtocolFlag, "ws", "The eth-client protocol")
	ServerCmd.Flags().StringVar(&ethHost, flags.EthHostFlag, "127.0.0.1", "The eth-client host")
	ServerCmd.Flags().IntVar(&ethPort, flags.EthPortFlag, 8546, "The eth-client port")

	ServerCmd.Flags().BoolVar(&metricsEnabled, metrics.MetricsEnabledFlag, false, "Enable metreics")
	ServerCmd.Flags().StringVar(&metricsHost, flags.MetricsHostFlag, "", "Metrics listening host")
	ServerCmd.Flags().IntVar(&metricsPort, flags.MetricsPortFlag, 9092, "Metrics listening port")
}
