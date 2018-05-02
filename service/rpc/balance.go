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
package rpc

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/getamis/sirius/log"
	"github.com/maichain/eth-indexer/service/pb"
	"github.com/maichain/mapi/api"
)

const (
	latestBlockNumber = -1
	ethToken          = "ETH"
)

func (s *server) GetBalance(ctx context.Context, req *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {
	logger := s.logger.New("trackingId", api.GetTrackingIDFromContext(ctx), "addr", req.Address, "number", req.BlockNumber, "token", req.Token)
	if req.BlockNumber < latestBlockNumber {
		log.Error("Invalid block number")
		return nil, ErrInvalidBlockNumber
	}

	res, err := s.getBalance(ctx, req.BlockNumber, req.Address, req.Token)
	if err != nil {
		logger.Error("Failed to get balance", "err", err)
		return nil, NewInternalServerError(err)
	}
	return res, nil
}

func (s *server) GetOffsetBalance(ctx context.Context, req *pb.GetOffsetBalanceRequest) (*pb.GetBalanceResponse, error) {
	logger := s.logger.New("trackingId", api.GetTrackingIDFromContext(ctx), "addr", req.Address, "offset", req.Offset, "token", req.Token)
	if req.Offset < 0 {
		log.Error("Invalid offset")
		return nil, ErrInvalidOffset
	}

	// Get latest block
	header, err := s.manager.FindLatestBlock()
	if err != nil {
		log.Error("Failed to get latest header", "err", err)
		return nil, NewInternalServerError(err)
	}

	// Get target block
	target := header.Number - req.Offset
	if target < 0 {
		log.Error("Offset is larger than current header number", "number", header.Number)
		return nil, ErrInvalidOffset
	}

	res, err := s.getBalance(ctx, target, req.Address, req.Token)
	if err != nil {
		logger.Error("Failed to get balance", "err", err)
		return nil, NewInternalServerError(err)
	}
	return res, nil
}

func (s *server) getBalance(ctx context.Context, blockNr int64, addr string, token string) (*pb.GetBalanceResponse, error) {
	// Get balance
	var err error
	var number *big.Int
	var balance *big.Int
	if token == ethToken {
		// Get Ether
		balance, number, err = s.manager.GetBalance(ctx, common.HexToAddress(addr), blockNr)
	} else {
		// Get ERC20 token
		balance, number, err = s.manager.GetERC20Balance(ctx, common.HexToAddress(token), common.HexToAddress(addr), blockNr)
	}
	if err != nil {
		return nil, err
	}
	return &pb.GetBalanceResponse{
		Amount:      balance.String(),
		BlockNumber: number.Int64(),
	}, nil
}
