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

	"github.com/ethereum/go-ethereum"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/getamis/eth-indexer/client"
	. "github.com/getamis/eth-indexer/service"
	"github.com/getamis/eth-indexer/service/pb"
	"github.com/getamis/eth-indexer/store"
	"github.com/getamis/sirius/log"
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
