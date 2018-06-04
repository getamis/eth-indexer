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
package erc20

import (
	"context"

	"github.com/ethereum/go-ethereum"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/getamis/sirius/log"
	"github.com/getamis/eth-indexer/client"
	. "github.com/getamis/eth-indexer/service"
	"github.com/getamis/eth-indexer/service/pb"
	"github.com/getamis/eth-indexer/store"
	"google.golang.org/grpc"
)

//go:generate mockery -dir ../pb -name ^ERC20Service
type server struct {
	manager     store.Manager
	client      client.EthClient
	middlewares []middleware
	logger      log.Logger
}

func New(manager store.Manager, client client.EthClient) *server {
	logger := log.New("erc20", "grpc")
	s := &server{
		manager: manager,
		client:  client,
		logger:  logger,
	}
	s.middlewares = append(s.middlewares,
		newValidatingMiddleware(logger, s),
	)
	return s
}

func (s *server) Bind(server *grpc.Server) {
	var srv pb.ERC20ServiceServer = s
	for _, mw := range s.middlewares {
		srv = mw(srv)
	}
	pb.RegisterERC20ServiceServer(server, srv)
}

func (s *server) Shutdown() {
	log.Info("ERC20 gRPC API shutdown successfully")
}

func (s *server) AddERC20(ctx context.Context, req *pb.AddERC20Request) (res *pb.AddERC20Response, err error) {
	logger := s.logger.New("address", req.Address, "blockNumber", req.BlockNumber)

	addr := ethCommon.HexToAddress(req.Address)
	erc20, err := s.client.GetERC20(ctx, addr, int64(req.BlockNumber))
	if err != nil {
		logger.Error("Failed to get code from ethereum", "err", err)
		if err == ethereum.NotFound {
			return nil, ErrInvalidAddress
		}
		return nil, ErrInternal
	}

	err = s.manager.InsertERC20(erc20)
	if err != nil {
		logger.Error("Failed to write ERC20 to db", "err", err)
		return nil, ErrInternal
	}

	return &pb.AddERC20Response{
		Address:     req.Address,
		BlockNumber: req.BlockNumber,
		TotalSupply: erc20.TotalSupply,
		Decimals:    int64(erc20.Decimals),
		Name:        erc20.Name,
	}, nil
}
