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
	"github.com/getamis/eth-indexer/client"
	"github.com/getamis/eth-indexer/common"
	"github.com/getamis/eth-indexer/model"
	"github.com/getamis/eth-indexer/store/account"
	"github.com/getamis/eth-indexer/store/subscription"
	"github.com/getamis/sirius/log"
)

type transferProcessor struct {
	logger      log.Logger
	blockNumber int64
	// tokenList includes ETH and erc20 tokens
	tokenList    map[gethCommon.Address]struct{}
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
	tokenList := make(map[gethCommon.Address]struct{}, len(erc20List)+1)
	tokenList[model.ETHAddress] = struct{}{}
	for addr := range erc20List {
		tokenList[gethCommon.HexToAddress(addr)] = struct{}{}
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

func (s *transferProcessor) process(ctx context.Context, events []*model.Transfer, coinbases []gethCommon.Address) (err error) {
	// Update total balance for new subscriptions, map[group][token]balance
	totalBalances := make(map[int64]map[gethCommon.Address]*big.Int)
	totalFees := make(map[int64]*big.Int)

	// Collect modified addresses
	seenAddrs := make(map[gethCommon.Address]struct{})
	var addrs [][]byte
	// From block / uncles coinbases
	for _, cb := range coinbases {
		if _, ok := seenAddrs[cb]; !ok {
			seenAddrs[cb] = struct{}{}
			addrs = append(addrs, cb.Bytes())
		}
	}
	// Collect fee payers
	// Note that we don't calculate fees from events because events with transfer value 0 (but still incur fee)
	// are not included in events.
	for _, tx := range s.txs {
		fromAddr := gethCommon.BytesToAddress(tx.From)
		if _, ok := seenAddrs[fromAddr]; !ok {
			seenAddrs[fromAddr] = struct{}{}
			addrs = append(addrs, tx.From)
		}
	}
	// From transfer events
	for _, e := range events {
		fromAddr := gethCommon.BytesToAddress(e.From)
		if _, ok := seenAddrs[fromAddr]; !ok {
			seenAddrs[fromAddr] = struct{}{}
			addrs = append(addrs, e.From)
		}
		toAddr := gethCommon.BytesToAddress(e.To)
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

	contractsAddrs := make(map[gethCommon.Address]map[gethCommon.Address]struct{})
	newSubs := make(map[gethCommon.Address]*model.Subscription)
	var newAddrs [][]byte
	for _, sub := range newSubResults {
		newAddr := gethCommon.BytesToAddress(sub.Address)
		newAddrs = append(newAddrs, sub.Address)
		newSubs[newAddr] = sub
		// Make sure to collect ETH/ERC20 balances for the new subscriptions too.
		for token := range s.tokenList {
			if contractsAddrs[token] == nil {
				contractsAddrs[token] = make(map[gethCommon.Address]struct{})
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

	// Construct a set of subscription for membership testing
	allSubs := make(map[gethCommon.Address]*model.Subscription)
	for _, sub := range subs {
		allSubs[gethCommon.BytesToAddress(sub.Address)] = sub
	}

	// Insert events if it's a subscribed account
	for _, e := range events {
		_, hasFrom := allSubs[gethCommon.BytesToAddress(e.From)]
		_, hasTo := allSubs[gethCommon.BytesToAddress(e.To)]
		if !hasFrom && !hasTo {
			continue
		}

		err := s.accountStore.InsertTransfer(e)
		if err != nil {
			s.logger.Error("Failed to insert ERC20 transfer event", "value", e.Value, "from", common.BytesToHex(e.From), "to", common.BytesToHex(e.To), "err", err)
			return err
		}
		contractAddr := gethCommon.BytesToAddress(e.Address)
		if contractsAddrs[contractAddr] == nil {
			contractsAddrs[contractAddr] = make(map[gethCommon.Address]struct{})
		}
		if hasFrom {
			from := gethCommon.BytesToAddress(e.From)
			contractsAddrs[contractAddr][from] = struct{}{}
		}
		if hasTo {
			to := gethCommon.BytesToAddress(e.To)
			contractsAddrs[contractAddr][to] = struct{}{}
		}
	}

	// Make sure coinbases are also included in balance queries to geth
	for _, addr := range coinbases {
		if allSubs[addr] == nil {
			continue
		}
		if contractsAddrs[model.ETHAddress] == nil {
			contractsAddrs[model.ETHAddress] = make(map[gethCommon.Address]struct{})
		}
		contractsAddrs[model.ETHAddress][addr] = struct{}{}
	}

	// Collect tx fee and make sure fee payers are included in balance queries to geth
	// Note that we don't calculate fees from events because events with transfer value 0 (but still incur fee)
	// are not included in events.
	feeDiff := make(map[gethCommon.Address]*big.Int)
	for i, tx := range s.txs {
		from := gethCommon.BytesToAddress(tx.From)
		if allSubs[from] == nil {
			continue
		}
		// Assume the tx and receipt are in the same order
		r := s.receipts[i]
		price, _ := new(big.Int).SetString(tx.GasPrice, 10)
		fee := new(big.Int).Mul(price, big.NewInt(int64(r.GasUsed)))
		if feeDiff[from] == nil {
			feeDiff[from] = new(big.Int).Set(fee)
		} else {
			feeDiff[from] = new(big.Int).Add(feeDiff[from], fee)
		}
		if contractsAddrs[model.ETHAddress] == nil {
			contractsAddrs[model.ETHAddress] = make(map[gethCommon.Address]struct{})
		}
		contractsAddrs[model.ETHAddress][from] = struct{}{}
	}

	// Get balances
	results, err := s.balancer.BalanceOf(ctx, big.NewInt(s.blockNumber), contractsAddrs)
	if err != nil {
		s.logger.Error("Failed to get ERC20 balance", "len", len(contractsAddrs), "err", err)
		return err
	}

	// Insert balance and calculate diff to total balances
	addrDiff := make(map[gethCommon.Address]map[gethCommon.Address]*big.Int)
	allAddrs := append(addrs, newAddrs...)
	for contractAddr, addrs := range results {
		// Get last recorded balance for these accounts
		latestBalances, err := s.getLatestBalances(contractAddr, allAddrs)
		if err != nil {
			s.logger.Error("Failed to get previous balances", "contractAddr", contractAddr.Hex(), "len", len(allAddrs), "err", err)
			return err
		}
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

			if addrDiff[contractAddr] == nil {
				addrDiff[contractAddr] = make(map[gethCommon.Address]*big.Int)
			}
			acct := latestBalances[addr]
			var diff *big.Int

			// If addr is a new subscription, add its balance to addrDiff for totalBalances.
			if newSubs[addr] != nil {
				// double check we don't have its previous balance
				if acct != nil {
					s.logger.Error("New subscription had previous balance", "block", acct.BlockNumber, "addr", addr.Hex(), "balance", acct.Balance)
					return common.ErrHasPrevBalance
				}
				diff = new(big.Int).Set(balance)
			} else {
				// make sure we have an old balance
				if acct == nil {
					s.logger.Error("Old subscription missing previous balance", "contractAddr", contractAddr.Hex(), "addr", addr.Hex())
					return common.ErrMissingPrevBalance
				}
				prevBalance, _ := new(big.Int).SetString(acct.Balance, 10)
				diff = new(big.Int).Sub(balance, prevBalance)
			}
			addrDiff[contractAddr][addr] = diff
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
				totalBalances[sub.Group] = make(map[gethCommon.Address]*big.Int)
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

// Get last recorded balance data for these accounts
func (s *transferProcessor) getLatestBalances(contractAddr gethCommon.Address, addrs [][]byte) (map[gethCommon.Address]*model.Account, error) {
	balances, err := s.accountStore.FindLatestAccounts(contractAddr, addrs)
	if err != nil {
		return nil, err
	}
	lastBalances := make(map[gethCommon.Address]*model.Account)
	for _, acct := range balances {
		lastBalances[gethCommon.BytesToAddress(acct.Address)] = acct
	}
	return lastBalances, nil
}
