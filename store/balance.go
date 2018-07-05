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
	"github.com/getamis/eth-indexer/model"
	"github.com/getamis/sirius/log"
	"github.com/shopspring/decimal"
)

var ErrInvalidBalance = errors.New("invalid balance")

func (srv *serviceManager) GetBalance(ctx context.Context, address gethCommon.Address, blockNr int64) (balance *big.Int, blockNumber *big.Int, err error) {
	logger := log.New("addr", address.Hex(), "number", blockNr)
	account, err := srv.FindAccount(model.ETHAddress, address, blockNr)
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
	blockNumber = big.NewInt(blockNr)
	return
}

func (srv *serviceManager) GetERC20Balance(ctx context.Context, contractAddress, address gethCommon.Address, blockNr int64) (*decimal.Decimal, *big.Int, error) {
	logger := log.New("contractAddr", contractAddress.Hex(), "addr", address.Hex(), "number", blockNr)
	// Find contract code
	erc20, err := srv.FindERC20(contractAddress)
	if err != nil {
		logger.Error("Failed to find erc20", "err", err)
		return nil, nil, err
	}

	account, err := srv.FindAccount(contractAddress, address, blockNr)
	if err != nil {
		logger.Error("Failed to find account", "err", err)
		return nil, nil, err
	}

	var ok bool
	balance, ok := new(big.Int).SetString(account.Balance, 10)
	if !ok {
		logger.Error("Failed to covert balance", "balance", account.Balance)
		return nil, nil, ErrInvalidBalance
	}
	result := decimal.NewFromBigInt(balance, -int32(erc20.Decimals))
	return &result, big.NewInt(blockNr), nil
}
