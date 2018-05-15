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
package store

import (
	"context"
	"errors"
	"math/big"

	gethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/getamis/sirius/log"
	"github.com/maichain/eth-indexer/common"
	"github.com/maichain/eth-indexer/model"
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
