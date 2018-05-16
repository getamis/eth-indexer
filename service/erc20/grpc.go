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
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/getamis/sirius/log"
	"github.com/maichain/eth-indexer/contracts"
	"github.com/maichain/eth-indexer/model"
	. "github.com/maichain/eth-indexer/service"
	"github.com/maichain/eth-indexer/service/indexer"
	"github.com/maichain/eth-indexer/service/pb"
	"github.com/maichain/eth-indexer/store"
	"google.golang.org/grpc"
)

//go:generate mockery -dir ../pb -name ^ERC20Service
type server struct {
	manager     store.Manager
	client      indexer.EthClient
	middlewares []middleware
	logger      log.Logger
}

func New(manager store.Manager, client indexer.EthClient) *server {
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
	blockNumber := big.NewInt(req.BlockNumber)

	// Get contract code
	code, err := s.client.CodeAt(ctx, addr, blockNumber)
	if err != nil {
		logger.Error("Failed to get code from ethereum", "err", err)
		if err == ethereum.NotFound {
			return nil, ErrInvalidAddress
		}
		return nil, ErrInternal
	}

	defer func() {
		if err != nil {
			return
		}
		// Write db if there is no error
		erc20 := &model.ERC20{
			BlockNumber: res.BlockNumber,
			Address:     addr.Bytes(),
			Code:        code,
			TotalSupply: res.TotalSupply,
			Decimals:    int(res.Decimals),
			Name:        res.Name,
		}
		err = s.manager.InsertERC20(erc20)
		if err != nil {
			logger.Error("Failed to write ERC20 to db", "err", err)
			err = ErrInternal
			res = nil
		}
	}()
	res = &pb.AddERC20Response{
		Address:     req.Address,
		BlockNumber: req.BlockNumber,
	}

	caller, err := contracts.NewERC20TokenCaller(addr, s.client)
	if err != nil {
		logger.Warn("Failed to initiate contract caller", "err", err)
		return res, nil
	}

	// Set decimals
	decimal, err := caller.Decimals(&bind.CallOpts{})
	if err != nil {
		logger.Warn("Failed to get decimals", "err", err)
	}
	res.Decimals = int64(decimal)

	// Set total supply
	supply, err := caller.TotalSupply(&bind.CallOpts{})
	if err != nil {
		logger.Warn("Failed to get total supply", "err", err)
	}
	res.TotalSupply = supply.String()

	// Set name
	name, err := caller.Name(&bind.CallOpts{})
	if err != nil {
		logger.Warn("Failed to get name", "err", err)
	}
	res.Name = name
	return res, nil
}
