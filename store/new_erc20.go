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
	"github.com/getamis/eth-indexer/client"
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
// We insert balances of ALL subscriptions and total balances for these new tokens.
func (m *manager) initNewERC20(ctx context.Context, accountStore account.Store, subStore subscription.Store, blockNumber int64, balancer client.Balancer) (map[gethCommon.Address]*model.ERC20, error) {
	// Get latest ERC20 list
	list, err := accountStore.ListNewERC20()
	if err != nil {
		return nil, err
	}
	// Return if there is no new ERC20
	if len(list) == 0 {
		return nil, nil
	}

	newTokens := make(map[gethCommon.Address]*model.ERC20, len(list))
	for _, e := range list {
		e.BlockNumber = blockNumber
		newTokens[gethCommon.BytesToAddress(e.Address)] = e
	}

	query := &model.QueryParameters{
		Page:  1,
		Limit: subLimit,
	}

	totalBalances := make(map[int64]map[gethCommon.Address]*big.Int)
	for {
		// List old subscriptions
		subs, total, err := subStore.ListOldSubscriptions(query)
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
		for _, sub := range subs {
			subMap[gethCommon.BytesToAddress(sub.Address)] = sub
		}
		// Contructs the requested balance map
		contractsAddrs := make(map[gethCommon.Address]map[gethCommon.Address]struct{})
		for _, token := range newTokens {
			contractsAddrs[gethCommon.BytesToAddress(token.Address)] = make(map[gethCommon.Address]struct{})
			for _, sub := range subs {
				contractsAddrs[gethCommon.BytesToAddress(token.Address)][gethCommon.BytesToAddress(sub.Address)] = struct{}{}
			}
		}
		// Get balances
		results, err := balancer.BalanceOf(ctx, big.NewInt(blockNumber), contractsAddrs)
		if err != nil {
			logger.Error("Failed to get ERC20 balance", "len", len(contractsAddrs), "err", err)
			return nil, err
		}

		// Update total balances
		for contractAddr, addrs := range results {
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
				err := accountStore.InsertAccount(b)
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

			err = subStore.InsertTotalBalance(tb)
			if err != nil {
				log.Error("Failed to insert total balance", "err", err)
				return nil, err
			}
		}
	}

	// Update erc20 tokens
	for _, token := range newTokens {
		err = accountStore.SetERC20Block(token)
		if err != nil {
			log.Error("Failed to set erc20 block number", "err", err)
			return nil, err
		}
	}
	return newTokens, nil
}
