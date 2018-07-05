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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/getamis/eth-indexer/client"
	"github.com/getamis/eth-indexer/model"
	"github.com/getamis/eth-indexer/store/account"
	"github.com/getamis/eth-indexer/store/subscription"
	"github.com/getamis/sirius/log"
)

type transferProcessor struct {
	logger      log.Logger
	blockNumber int64
	// tokenList includes ETH and erc20 tokens
	tokenList    map[common.Address]struct{}
	subStore     subscription.Store
	accountStore account.Store
	balancer     client.Balancer

	// Used to calculate transaction fee
	receipts []*types.Receipt
	txs      []*model.Transaction
}

func newTransferProcessor(blockNumber int64,
	erc20List map[string]*model.ERC20,
	receipts []*types.Receipt,
	txs []*model.Transaction,
	subStore subscription.Store,
	accountStore account.Store,
	balancer client.Balancer) *transferProcessor {
	tokenList := make(map[common.Address]struct{}, len(erc20List)+1)
	tokenList[model.ETHAddress] = struct{}{}
	for addr := range erc20List {
		tokenList[common.HexToAddress(addr)] = struct{}{}
	}
	return &transferProcessor{
		logger:       log.New("number", blockNumber),
		blockNumber:  blockNumber,
		tokenList:    tokenList,
		subStore:     subStore,
		accountStore: accountStore,
		balancer:     balancer,
		receipts:     receipts,
		txs:          txs,
	}
}

func (s *transferProcessor) process(ctx context.Context, events []*model.Transfer) (err error) {
	// Update total balance for new subscriptions, map[group][token]balance
	totalBalances := make(map[int64]map[common.Address]*big.Int)
	totalFees := make(map[int64]*big.Int)

	// Collect modified addresses
	seenAddrs := make(map[common.Address]struct{})
	var addrs [][]byte
	for _, e := range events {
		fromAddr := common.BytesToAddress(e.From)
		if _, ok := seenAddrs[fromAddr]; !ok {
			seenAddrs[fromAddr] = struct{}{}
			addrs = append(addrs, e.From)
		}
		toAddr := common.BytesToAddress(e.To)
		if _, ok := seenAddrs[toAddr]; !ok {
			seenAddrs[toAddr] = struct{}{}
			addrs = append(addrs, e.To)
		}
	}

	// Add new subscriptions
	newSubResults, err := s.subStore.Find(0)
	if err != nil {
		s.logger.Error("Failed to find subscriptions", "err", err)
		return err
	}

	contractsAddrs := make(map[common.Address]map[common.Address]struct{})
	newSubs := make(map[common.Address]*model.Subscription)
	var newAddrs [][]byte
	for _, sub := range newSubResults {
		newAddr := common.BytesToAddress(sub.Address)
		newAddrs = append(newAddrs, sub.Address)
		newSubs[newAddr] = sub
		// Make sure to collect ETH/ERC20 balances for the new subscriptions too.
		for token := range s.tokenList {
			if contractsAddrs[token] == nil {
				contractsAddrs[token] = make(map[common.Address]struct{})
			}
			contractsAddrs[token][newAddr] = struct{}{}
		}
	}

	// Get subscribed accounts whose balances are changed, including the new subscriptions
	subs, err := s.subStore.FindOldSubscriptions(addrs)
	if err != nil {
		s.logger.Error("Failed to find subscription address", "len", len(addrs), "err", err)
		return err
	}

	// Add new subscriptions to the processing list
	subs = append(subs, newSubResults...)
	if len(subs) == 0 {
		s.logger.Debug("No modified or newly subscribed accounts")
		return
	}

	// Calculate tx fee
	fees := make(map[common.Hash]*big.Int)
	// Assume the tx and receipt are in the same order
	for i, tx := range s.txs {
		r := s.receipts[i]
		price, _ := new(big.Int).SetString(tx.GasPrice, 10)
		fees[common.BytesToHash(tx.Hash)] = new(big.Int).Mul(price, big.NewInt(int64(r.GasUsed)))
	}

	// Construct a set of subscription for membership testing
	allSubs := make(map[common.Address]*model.Subscription)
	for _, sub := range subs {
		allSubs[common.BytesToAddress(sub.Address)] = sub
	}

	// Insert events if it's a subscribed account
	addrDiff := make(map[common.Address]map[common.Address]*big.Int)
	feeDiff := make(map[common.Address]*big.Int)
	for _, e := range events {
		_, hasFrom := allSubs[common.BytesToAddress(e.From)]
		_, hasTo := allSubs[common.BytesToAddress(e.To)]
		if !hasFrom && !hasTo {
			continue
		}

		err := s.accountStore.InsertTransfer(e)
		if err != nil {
			s.logger.Error("Failed to insert ERC20 transfer event", "value", e.Value, "from", common.Bytes2Hex(e.From), "to", common.Bytes2Hex(e.To), "err", err)
			return err
		}
		contractAddr := common.BytesToAddress(e.Address)
		if addrDiff[contractAddr] == nil {
			addrDiff[contractAddr] = make(map[common.Address]*big.Int)
			contractsAddrs[contractAddr] = make(map[common.Address]struct{})
		}
		d, _ := new(big.Int).SetString(e.Value, 10)
		if hasFrom {
			from := common.BytesToAddress(e.From)
			if addrDiff[contractAddr][from] == nil {
				addrDiff[contractAddr][from] = new(big.Int).Neg(d)
				contractsAddrs[contractAddr][from] = struct{}{}
			} else {
				addrDiff[contractAddr][from] = new(big.Int).Add(addrDiff[contractAddr][from], new(big.Int).Neg(d))
			}

			// Add fee if it's a ETH event
			if f, ok := fees[common.BytesToHash(e.TxHash)]; ok && bytes.Equal(e.Address, model.ETHBytes) {
				if feeDiff[from] == nil {
					feeDiff[from] = new(big.Int).Set(f)
				} else {
					feeDiff[from] = new(big.Int).Add(feeDiff[from], f)
				}
			}
		}
		if hasTo {
			to := common.BytesToAddress(e.To)
			if addrDiff[contractAddr][to] == nil {
				addrDiff[contractAddr][to] = d
				contractsAddrs[contractAddr][to] = struct{}{}
			} else {
				addrDiff[contractAddr][to] = new(big.Int).Add(addrDiff[contractAddr][to], d)
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

			// If addr is a new subscription, add its balance to addrDiff for totalBalances.
			// Note that we overwrite the original value if it exists, because the diff to
			// totalBalances for a new subscription is its current balance.
			if newSubs[addr] != nil {
				if addrDiff[contractAddr] == nil {
					addrDiff[contractAddr] = make(map[common.Address]*big.Int)
				}
				addrDiff[contractAddr][addr] = balance
			}
		}
	}

	// Update the subscriptions table for the new subscriptions
	err = s.subStore.BatchUpdateBlockNumber(s.blockNumber, newAddrs)
	if err != nil {
		s.logger.Error("Failed to update block number", "err", err)
		return err
	}

	// Add diff in total balances
	for token, addrs := range addrDiff {
		for addr, d := range addrs {
			sub, ok := allSubs[addr]
			if !ok {
				s.logger.Warn("Missing address from all subscriptions", "addr", addr.Hex())
				continue
			}

			// Init total balance for the group
			if totalBalances[sub.Group] == nil {
				totalBalances[sub.Group] = make(map[common.Address]*big.Int)
			}
			tb, ok := totalBalances[sub.Group][token]
			if !ok {
				b, err := s.subStore.FindTotalBalance(s.blockNumber-1, token, sub.Group)
				if err != nil {
					s.logger.Error("Failed to find total balance", "err", err)
					return err
				}

				tb, _ = new(big.Int).SetString(b.Balance, 10)
			}
			totalBalances[sub.Group][token] = new(big.Int).Add(tb, d)

			// Consider tx fees
			if f, ok := feeDiff[addr]; ok && token == model.ETHAddress {
				if totalFees[sub.Group] == nil {
					totalFees[sub.Group] = new(big.Int).Set(f)
				} else {
					totalFees[sub.Group] = new(big.Int).Add(f, totalFees[sub.Group])
				}

				// Subtract tx fee from total balance if it's not a new subscriptions.
				if newSubs[addr] == nil {
					totalBalances[sub.Group][token] = new(big.Int).Sub(totalBalances[sub.Group][token], f)
				}
			}
		}
	}

	for group, addrs := range totalBalances {
		for token, d := range addrs {
			tb := &model.TotalBalance{
				Token:       token.Bytes(),
				BlockNumber: s.blockNumber,
				Group:       group,
				TxFee:       "0",
				Balance:     d.String(),
			}

			if f, ok := totalFees[group]; ok && token == model.ETHAddress {
				tb.TxFee = f.String()
			}
			err = s.subStore.InsertTotalBalance(tb)
			if err != nil {
				return
			}
		}
	}
	return nil
}
