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
	"bytes"
	"context"
	"math/big"

	gethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/getamis/eth-indexer/client"
	"github.com/getamis/eth-indexer/common"
	"github.com/getamis/eth-indexer/model"
	"github.com/getamis/eth-indexer/store/account"
	subscriptionStore "github.com/getamis/eth-indexer/store/subscription"
	"github.com/getamis/sirius/log"
)

type subscription struct {
	logger      log.Logger
	blockNumber int64
	// tokenList includes ETH and erc20 tokens
	tokenList         map[gethCommon.Address]struct{}
	subscriptionStore subscriptionStore.Store
	accountStore      account.Store
	balancer          client.Balancer

	//
	newSubs     map[gethCommon.Address]*model.Subscription
	newBalances map[gethCommon.Address]map[gethCommon.Address]*big.Int
}

func newSubscription(blockNumber int64,
	erc20List map[string]*model.ERC20,
	subscriptionStore subscriptionStore.Store,
	accountStore account.Store,
	balancer client.Balancer) *subscription {
	tokenList := make(map[gethCommon.Address]struct{}, len(erc20List)+1)
	tokenList[model.ETHAddress] = struct{}{}
	for addr := range erc20List {
		tokenList[gethCommon.HexToAddress(addr)] = struct{}{}
	}
	return &subscription{
		logger:            log.New("number", blockNumber),
		blockNumber:       blockNumber,
		tokenList:         tokenList,
		subscriptionStore: subscriptionStore,
		accountStore:      accountStore,
		balancer:          balancer,
	}
}

func (s *subscription) init(ctx context.Context) error {
	// Get all new subscriptions
	subs, err := s.subscriptionStore.Find(0)
	if err != nil {
		s.logger.Error("Failed to find subscriptions", "err", err)
		return err
	}

	// Return if no new subscriptions
	if len(subs) == 0 {
		return nil
	}

	blockNumber := big.NewInt(s.blockNumber)
	// Construct the requested addresses
	contractsAddrs := make(map[gethCommon.Address]map[gethCommon.Address]struct{}, len(s.tokenList))
	// Add ERC20 contract addresses
	for addr := range s.tokenList {
		contractsAddrs[addr] = make(map[gethCommon.Address]struct{}, len(subs))
		for _, sub := range subs {
			contractsAddrs[addr][gethCommon.BytesToAddress(sub.Address)] = struct{}{}
		}
	}

	// Get balances of all new subscriptions
	s.newBalances, err = s.balancer.BalanceOf(ctx, blockNumber, contractsAddrs)
	if err != nil {
		s.logger.Error("Failed to get ERC20 balance", "len", len(s.newBalances), "err", err)
		return err
	}

	// Update balances
	for token, addrs := range s.newBalances {
		for addr, balance := range addrs {
			err = s.accountStore.InsertAccount(&model.Account{
				ContractAddress: token.Bytes(),
				BlockNumber:     s.blockNumber,
				Address:         addr.Bytes(),
				Balance:         balance.String(),
			})
			if err != nil {
				s.logger.Error("Failed to insert ERC20 account", "err", err)
				return err
			}
		}
	}

	// Update subscription table
	for _, sub := range subs {
		sub.BlockNumber = s.blockNumber
		err = s.subscriptionStore.UpdateBlockNumber(sub)
		if err != nil {
			s.logger.Error("Failed to update block number", "err", err)
			return err
		}
	}

	// Construct the new subs in map
	s.newSubs = make(map[gethCommon.Address]*model.Subscription)
	for _, sub := range subs {
		s.newSubs[gethCommon.BytesToAddress(sub.Address)] = sub
	}

	return nil
}

func (s *subscription) insert(ctx context.Context, events []*model.Transfer) (err error) {
	// Update total balance for new subscriptions, map[group][token]balance
	totalBalances := make(map[int64]map[gethCommon.Address]*big.Int)
	for _, sub := range s.newSubs {
		if totalBalances[sub.Group] == nil {
			totalBalances[sub.Group] = make(map[gethCommon.Address]*big.Int)
		}

		for token := range s.tokenList {
			b, err := s.subscriptionStore.FindTotalBalance(s.blockNumber-1, token, sub.Group)
			if err != nil {
				s.logger.Error("Failed to find total balance", "err", err)
				return err
			}

			d, _ := new(big.Int).SetString(b.Balance, 10)
			addr := gethCommon.BytesToAddress(sub.Address)
			totalBalances[sub.Group][token] = new(big.Int).Add(d, s.newBalances[token][addr])
		}
	}

	// Insert total balances
	defer func() {
		for group, addrs := range totalBalances {
			for token, d := range addrs {
				err = s.subscriptionStore.InsertTotalBalance(&model.TotalBalance{
					Token:       token.Bytes(),
					BlockNumber: s.blockNumber,
					Group:       group,
					Balance:     d.String(),
				})
				if err != nil {
					return
				}
			}
		}
	}()

	// Collect modified addresses
	mapAddrs := make(map[string]struct{})
	for _, e := range events {
		// Exclude new subscriptions
		if _, ok := s.newSubs[gethCommon.BytesToAddress(e.From)]; !ok {
			mapAddrs[common.BytesToHex(e.From)] = struct{}{}
		}
		if _, ok := s.newSubs[gethCommon.BytesToAddress(e.To)]; !ok {
			mapAddrs[common.BytesToHex(e.To)] = struct{}{}
		}
	}
	var addrs [][]byte
	for addr := range mapAddrs {
		addrs = append(addrs, common.HexToBytes(addr))
	}

	// Get subscribed accounts whose balances are chanaged
	subs, err := s.subscriptionStore.FindByAddresses(addrs)
	if err != nil {
		s.logger.Error("Failed to find subscription address", "len", len(addrs), "err", err)
		return err
	}
	if len(subs) == 0 {
		s.logger.Debug("No subscribed accounts")
		return
	}

	// Insert events if it's a subscribed account
	addrDiff := make(map[gethCommon.Address]map[gethCommon.Address]*big.Int)
	contractsAddrs := make(map[gethCommon.Address]map[gethCommon.Address]struct{})
	for _, sub := range subs {
		for _, e := range events {
			if !bytes.Equal(e.From, sub.Address) && !bytes.Equal(e.To, sub.Address) {
				continue
			}

			err := s.accountStore.InsertTransfer(e)
			if err != nil {
				s.logger.Error("Failed to insert ERC20 transfer event", "value", e.Value, "from", gethCommon.Bytes2Hex(e.From), "to", gethCommon.Bytes2Hex(e.To), "err", err)
				return err
			}
			contractAddr := gethCommon.BytesToAddress(e.Address)
			if addrDiff[contractAddr] == nil {
				addrDiff[contractAddr] = make(map[gethCommon.Address]*big.Int)
				contractsAddrs[contractAddr] = make(map[gethCommon.Address]struct{})
			}
			d, _ := new(big.Int).SetString(e.Value, 10)
			if bytes.Equal(e.From, sub.Address) {
				if addrDiff[contractAddr][gethCommon.BytesToAddress(e.From)] == nil {
					addrDiff[contractAddr][gethCommon.BytesToAddress(e.From)] = new(big.Int).Neg(d)
					contractsAddrs[contractAddr][gethCommon.BytesToAddress(e.From)] = struct{}{}
				} else {
					addrDiff[contractAddr][gethCommon.BytesToAddress(e.From)] = new(big.Int).Add(addrDiff[contractAddr][gethCommon.BytesToAddress(e.From)], new(big.Int).Neg(d))
				}
			}
			if bytes.Equal(e.To, sub.Address) {
				if addrDiff[contractAddr][gethCommon.BytesToAddress(e.To)] == nil {
					addrDiff[contractAddr][gethCommon.BytesToAddress(e.To)] = d
					contractsAddrs[contractAddr][gethCommon.BytesToAddress(e.To)] = struct{}{}
				} else {
					addrDiff[contractAddr][gethCommon.BytesToAddress(e.To)] = new(big.Int).Add(addrDiff[contractAddr][gethCommon.BytesToAddress(e.To)], d)
				}
			}
		}
	}

	// Get balances
	results, err := s.balancer.BalanceOf(ctx, big.NewInt(s.blockNumber), contractsAddrs)
	if err != nil {
		s.logger.Error("Failed to get ERC20 balance", "len", len(contractsAddrs), "err", err)
		return err
	}

	// Insert balance if it's a subscribed account
	for contractAddr, addrs := range results {
		for addr, balance := range addrs {
			b := &model.Account{
				ContractAddress: contractAddr.Bytes(),
				BlockNumber:     s.blockNumber,
				Address:         addr.Bytes(),
				Balance:         balance.String(),
			}
			err := s.accountStore.InsertAccount(b)
			if err != nil {
				s.logger.Error("Failed to insert ERC20 account", "err", err)
				return err
			}
		}
	}

	// Add diff in total balances
	for _, sub := range subs {
		addr := gethCommon.BytesToAddress(sub.Address)
		for token, addrs := range addrDiff {
			d, ok := addrs[addr]
			if !ok {
				continue
			}

			// Init total balance for the group
			if totalBalances[sub.Group] == nil {
				totalBalances[sub.Group] = make(map[gethCommon.Address]*big.Int)
			}
			tb, ok := totalBalances[sub.Group][token]
			if !ok {
				b, err := s.subscriptionStore.FindTotalBalance(s.blockNumber-1, token, sub.Group)
				if err != nil {
					s.logger.Error("Failed to find total balance", "err", err)
					return err
				}

				tb, _ = new(big.Int).SetString(b.Balance, 10)
			}
			totalBalances[sub.Group][token] = new(big.Int).Add(tb, d)
		}
	}

	return nil
}
