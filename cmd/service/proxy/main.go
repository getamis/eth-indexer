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

package proxy

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/getamis/sirius/log"
	"github.com/getamis/sirius/rpc"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"github.com/getamis/eth-indexer/service/proxy"
)

var (
	host string
	port int

	gRPCHost string
	gRPCPort int

	corsOrigins     []string
	corsCredentials bool
)

// ServerCmd represents the proxy command
var ServerCmd = &cobra.Command{
	Use:   "proxy",
	Short: "proxy runs a HTTP reverse proxy which translates a RESTful JSON API into gRPC",
	Long:  `proxy runs a HTTP reverse proxy which translates a RESTful JSON API into gRPC.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		gRPCEndpoint := fmt.Sprintf("%s:%d", gRPCHost, gRPCPort)
		opts := []grpc.DialOption{
			grpc.WithInsecure(),
		}

		l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
		if err != nil {
			log.Error("Failed to listen", "host", host, "port", port, "err", err)
			return err
		}

		s := rpc.NewProxy(
			rpc.Proxies(
				proxy.NewProxy(gRPCEndpoint, opts...),
			),
		)

		go func() {
			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
			defer signal.Stop(sigs)
			log.Debug("Shutting down", "signal", <-sigs)
			s.Shutdown()
		}()

		log.Info("Starting WS proxy", "host", host, "port", port)

		if err := s.Serve(l); err != http.ErrServerClosed {
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
	// ServerCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	ServerCmd.Flags().StringVar(&host, "host", "", "The http server listening host")
	ServerCmd.Flags().IntVar(&port, "port", 8080, "The http server listening port")
	ServerCmd.Flags().StringVar(&gRPCHost, "grpc.host", "", "The gRPC server listening host")
	ServerCmd.Flags().IntVar(&gRPCPort, "grpc.port", 5487, "The gRPC server listening port")
	ServerCmd.Flags().StringSliceVar(&corsOrigins, "cors.origins", []string{}, "The domains are allowed for Cross-Origin Resource Sharing (CORS)")
	ServerCmd.Flags().BoolVar(&corsCredentials, "cors.credentials", false, "Allow Access-Control-Allow-Credentials or not")

	// ServerCmd.Flags().BoolVar(&insecure, "insecure", false, "Disable header validation")
}
