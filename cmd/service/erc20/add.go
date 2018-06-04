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

package erc20

import (
	"context"
	"fmt"

	"github.com/getamis/eth-indexer/cmd/flags"
	"github.com/getamis/eth-indexer/service/pb"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

var (
	host string
	port int

	address     string
	blockNumber int
)

var AddCmd = &cobra.Command{
	Use:   "add",
	Short: "erc20 sends add erc20 token request to rpc",
	Long:  `erc20 sends add erc20 token request to rpc`,
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, err := grpc.Dial(fmt.Sprintf("%s:%d", host, port), grpc.WithInsecure())
		if err != nil {
			return err
		}

		client := pb.NewERC20ServiceClient(conn)
		ctx := context.Background()
		res, err := client.AddERC20(ctx, &pb.AddERC20Request{
			Address:     address,
			BlockNumber: int64(blockNumber),
		})
		if err != nil {
			return err
		}
		fmt.Printf("ERC20 contract is added, address = %v, block number = %v, name = %v, decimals = %v, total supply = %v", res.Address, res.BlockNumber, res.Name, res.Decimals, res.TotalSupply)
		return nil
	},
}

func init() {
	AddCmd.Flags().StringVar(&host, flags.Host, "", "The gRPC server listening host")
	AddCmd.Flags().IntVar(&port, flags.Port, 5487, "The gRPC server listening port")

	AddCmd.Flags().StringVar(&address, "address", "", "ERC20 contract address")
	AddCmd.Flags().IntVar(&blockNumber, "block-number", -1, "The block number which the ERC20 contract is deployed")
}
