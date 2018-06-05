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
