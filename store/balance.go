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

package store

import (
	"context"
	"errors"
	"math/big"

	gethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/getamis/eth-indexer/common"
	"github.com/getamis/eth-indexer/model"
	"github.com/getamis/sirius/log"
)

var ErrInvalidBalance = errors.New("invalid balance")

func (srv *serviceManager) GetBalance(ctx context.Context, address gethCommon.Address, blockNr int64) (balance *big.Int, blockNumber *big.Int, err error) {
	logger := log.New("addr", address.Hex(), "number", blockNr)
	// Find header
	var hdr *model.Header
	if common.IsLatestBlock(blockNr) {
		hdr, err = srv.FindLatestBlock()
	} else {
		hdr, err = srv.FindBlockByNumber(blockNr)
	}
	if err != nil {
		logger.Error("Failed to find header for block", "err", err)
		return nil, nil, err
	}
	blockNumber = big.NewInt(hdr.Number)

	// Find account
	account, err := srv.FindAccount(address, hdr.Number)
	if err != nil {
		logger.Error("Failed to find account", "err", err)
		return nil, nil, err
	}
	var ok bool
	balance, ok = new(big.Int).SetString(account.Balance, 10)
	if !ok {
		logger.Error("Failed to covert balance", "balance", account.Balance)
		return nil, nil, ErrInvalidBalance
	}

	return
}
