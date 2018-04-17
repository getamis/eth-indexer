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

	var number *big.Int
	var balance *big.Int
	var err error
	if req.Token == ethToken {
		// Get Ether
		balance, number, err = s.accountAPI.GetBalance(ctx, common.HexToAddress(req.Address), req.BlockNumber)
	} else {
		// Get ERC20 token
		balance, number, err = s.accountAPI.GetERC20Balance(ctx, common.HexToAddress(req.Token), common.HexToAddress(req.Address), req.BlockNumber)
	}
	if err != nil {
		logger.Error("Failed to get balance", "err", err)
		return nil, NewInternalServerError(err)
	}
	return &pb.GetBalanceResponse{
		Amount:      balance.String(),
		BlockNumber: number.Int64(),
	}, nil
}
