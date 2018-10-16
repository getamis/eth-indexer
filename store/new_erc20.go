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
	"math/big"

	gethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/getamis/eth-indexer/model"
	"github.com/getamis/eth-indexer/store/account"
	"github.com/getamis/eth-indexer/store/subscription"
	"github.com/getamis/sirius/log"
)

var (
	// subLimit defines the limit of old subscription query
	subLimit = uint64(1000)
)

// initNewERC20 inits all new erc20 tokens.
// 1. insert balances of ALL subscriptions and total balances for these new tokens at the given block.
// 2. return new tokens with the next block number
func (m *manager) initNewERC20(ctx context.Context, accountStore account.Store, subStore subscription.Store, block *types.Block) (map[gethCommon.Address]*model.ERC20, error) {
	blockNumber := block.Number().Int64()
	blockHash := block.Hash()

	// Get latest ERC20 list
	list, err := accountStore.ListNewERC20(ctx)
	if err != nil {
		return nil, err
	}
	// Return if there is no new ERC20
	if len(list) == 0 {
		return nil, nil
	}

	nextBlockNumber := blockNumber + 1
	newTokens := make(map[gethCommon.Address]*model.ERC20, len(list))
	for _, e := range list {
		e.BlockNumber = nextBlockNumber
		newTokens[gethCommon.BytesToAddress(e.Address)] = e
		log.Debug("Try to add new ERC20 token", "name", e.Name, "addr", gethCommon.BytesToAddress(e.Address).Hex())
	}

	query := &model.QueryParameters{
		Page:  1,
		Limit: subLimit,
	}

	totalBalances := make(map[int64]map[gethCommon.Address]*big.Int)
	for {
		// List old subscriptions
		subs, total, err := subStore.ListOldSubscriptions(ctx, query)
		logger := log.New("total", total, "page", query.Page, "limit", query.Limit)
		if err != nil {
			logger.Error("Failed to list old subscriptions", "err", err)
			return nil, err
		}
		if len(subs) == 0 {
			logger.Debug("No more old subscriptions")
			break
		}

		// Construct a set of subscription for membership testing
		subMap := make(map[gethCommon.Address]*model.Subscription)
		balancesByContracts := make(map[gethCommon.Address]map[gethCommon.Address]*big.Int)
		for _, sub := range subs {
			subAddr := gethCommon.BytesToAddress(sub.Address)
			subMap[subAddr] = sub
			for _, token := range newTokens {
				tokenAddr := gethCommon.BytesToAddress(token.Address)
				if balancesByContracts[tokenAddr] == nil {
					balancesByContracts[tokenAddr] = make(map[gethCommon.Address]*big.Int)
				}
				balancesByContracts[tokenAddr][subAddr] = new(big.Int)
			}
		}
		// Get balances
		err = m.balancer.BalanceOf(ctx, blockHash, balancesByContracts)
		if err != nil {
			logger.Error("Failed to get ERC20 balance", "len", len(balancesByContracts), "err", err)
			return nil, err
		}

		// Update total balances
		for contractAddr, addrs := range balancesByContracts {
			for addr, balance := range addrs {
				sub, ok := subMap[addr]
				if !ok {
					logger.Warn("Missing address from all subscriptions", "addr", addr.Hex())
					continue
				}

				// Insert balances
				b := &model.Account{
					ContractAddress: contractAddr.Bytes(),
					BlockNumber:     blockNumber,
					Address:         addr.Bytes(),
					Balance:         balance.String(),
				}
				err := accountStore.InsertAccount(ctx, b)
				if err != nil {
					logger.Error("Failed to insert ERC20 account", "err", err)
					return nil, err
				}

				// Init total balance for the group
				if totalBalances[sub.Group] == nil {
					totalBalances[sub.Group] = make(map[gethCommon.Address]*big.Int)
				}
				if totalBalances[sub.Group][contractAddr] == nil {
					totalBalances[sub.Group][contractAddr] = new(big.Int).Set(balance)
				} else {
					totalBalances[sub.Group][contractAddr] = new(big.Int).Add(totalBalances[sub.Group][contractAddr], balance)
				}
			}
		}

		if query.Page*query.Limit >= total {
			logger.Debug("No more old subscriptions")
			break
		}
		query.Page++
	}

	// Insert total balances
	for group, addrs := range totalBalances {
		for token, d := range addrs {
			tb := &model.TotalBalance{
				Token:       token.Bytes(),
				BlockNumber: blockNumber,
				Group:       group,
				TxFee:       "0",
				Balance:     d.String(),
			}

			err = subStore.InsertTotalBalance(ctx, tb)
			if err != nil {
				log.Error("Failed to insert total balance", "err", err)
				return nil, err
			}
		}
	}

	// Update erc20 tokens to the next block number
	var addrs [][]byte
	for _, token := range newTokens {
		addrs = append(addrs, token.Address)
	}
	err = accountStore.BatchUpdateERC20BlockNumber(ctx, nextBlockNumber, addrs)
	if err != nil {
		log.Error("Failed to update erc20 block number", "err", err)
		return nil, err
	}
	return newTokens, nil
}
