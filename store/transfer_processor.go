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

var (
	// estNumDiffAcct defines the estimated number of changed accounts in this block
	estNumDiffAcct = 10
	// newSubscriptionLimit defines the number of new subscriptions processed for each block, hoping to keep total number of accounts sent to geth in a single batch
	newSubscriptionLimit = uint64(client.ChunkSize - estNumDiffAcct)
)

type transferProcessor struct {
	logger      log.Logger
	blockNumber int64
	blockHash   gethCommon.Hash
	// tokenList includes ETH and erc20 tokens
	tokenList    map[gethCommon.Address]*model.ERC20
	subStore     subscription.Store
	accountStore account.Store
	balancer     client.Balancer

	// Used to calculate transaction fee
	receipts []*types.Receipt
	txs      []*model.Transaction
}

func newTransferProcessor(block *types.Block,
	tokenList map[gethCommon.Address]*model.ERC20,
	receipts []*types.Receipt,
	txs []*model.Transaction,
	subStore subscription.Store,
	accountStore account.Store,
	balancer client.Balancer) *transferProcessor {

	return &transferProcessor{
		logger:       log.New("number", block.NumberU64()),
		blockNumber:  block.Number().Int64(),
		blockHash:    block.Hash(),
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
	totalBalances := make(map[int64]map[gethCommon.Address]*big.Int)
	totalFees := make(map[int64]*big.Int)
	totalMinerReward := make(map[int64]*big.Int)
	totalUncleRewards := make(map[int64]*big.Int)

	// Collect modified addresses
	seenAddrs := make(map[gethCommon.Address]struct{})
	var addrs [][]byte
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
	// From transfer events (including miner and uncles)
	for _, e := range events {
		fromAddr := gethCommon.BytesToAddress(e.From)
		if _, ok := seenAddrs[fromAddr]; !ok {
			if !e.IsMinerRewardEvent() && !e.IsUncleRewardEvent() {
				seenAddrs[fromAddr] = struct{}{}
				addrs = append(addrs, e.From)
			}
		}
		toAddr := gethCommon.BytesToAddress(e.To)
		if _, ok := seenAddrs[toAddr]; !ok {
			seenAddrs[toAddr] = struct{}{}
			addrs = append(addrs, e.To)
		}
	}
	// Add new subscriptions
	newSubResults, total, err := s.subStore.Find(ctx, 0, &model.QueryParameters{
		Page:  1,
		Limit: newSubscriptionLimit,
	})
	if err != nil {
		s.logger.Error("Failed to find subscriptions", "err", err)
		return err
	}
	s.logger.Trace("Find new subscriptions", "handled", len(newSubResults), "total", total)

	balancesByContracts := make(map[gethCommon.Address]map[gethCommon.Address]*big.Int)
	newSubs := make(map[gethCommon.Address]*model.Subscription)
	var newAddrs [][]byte
	for _, sub := range newSubResults {
		newAddr := gethCommon.BytesToAddress(sub.Address)
		newAddrs = append(newAddrs, sub.Address)
		newSubs[newAddr] = sub
		// Make sure to collect ETH/ERC20 balances for the new subscriptions too.
		for token := range s.tokenList {
			if balancesByContracts[token] == nil {
				balancesByContracts[token] = make(map[gethCommon.Address]*big.Int)
			}
			balancesByContracts[token][newAddr] = new(big.Int)
		}
	}

	// Get subscribed accounts whose balances are changed, including the new subscriptions
	subs, err := s.subStore.FindOldSubscriptions(ctx, addrs)
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

	minerRewardDiff := make(map[gethCommon.Address]*big.Int)
	uncleRewardDiff := make(map[gethCommon.Address]*big.Int)
	// Insert events and calculate miner, uncle reward if it's a subscribed account
	for _, e := range events {
		_, hasFrom := allSubs[gethCommon.BytesToAddress(e.From)]
		_, hasTo := allSubs[gethCommon.BytesToAddress(e.To)]
		if !hasFrom && !hasTo {
			continue
		}

		err := s.accountStore.InsertTransfer(ctx, e)
		if err != nil {
			s.logger.Error("Failed to insert ERC20 transfer event", "value", e.Value, "from", common.BytesToHex(e.From), "to", common.BytesToHex(e.To), "err", err)
			return err
		}
		contractAddr := gethCommon.BytesToAddress(e.Address)
		if balancesByContracts[contractAddr] == nil {
			balancesByContracts[contractAddr] = make(map[gethCommon.Address]*big.Int)
		}
		if hasFrom {
			from := gethCommon.BytesToAddress(e.From)
			balancesByContracts[contractAddr][from] = new(big.Int)
		}
		if hasTo {
			to := gethCommon.BytesToAddress(e.To)
			balancesByContracts[contractAddr][to] = new(big.Int)
		}

		if e.IsMinerRewardEvent() {
			to := gethCommon.BytesToAddress(e.To)
			reward, _ := new(big.Int).SetString(e.Value, 10)

			if len(minerRewardDiff) > 1 {
				s.printUnexpectedRewardEvent(e, minerRewardDiff)
				return model.ErrTooManyMiners
			}
			minerRewardDiff[to] = new(big.Int).Set(reward)
		} else if e.IsUncleRewardEvent() {
			if len(uncleRewardDiff) > model.MaxUncles {
				s.printUnexpectedRewardEvent(e, uncleRewardDiff)
				return model.ErrTooManyUncles
			}
			to := gethCommon.BytesToAddress(e.To)
			reward, _ := new(big.Int).SetString(e.Value, 10)
			if uncleRewardDiff[to] == nil {
				uncleRewardDiff[to] = new(big.Int).Set(reward)
			} else {
				uncleRewardDiff[to].Add(uncleRewardDiff[to], reward)
			}
		}
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
		price := big.NewInt(tx.GasPrice)
		fee := new(big.Int).Mul(price, big.NewInt(int64(r.GasUsed)))
		if feeDiff[from] == nil {
			feeDiff[from] = new(big.Int).Set(fee)
		} else {
			feeDiff[from] = new(big.Int).Add(feeDiff[from], fee)
		}
		if balancesByContracts[model.ETHAddress] == nil {
			balancesByContracts[model.ETHAddress] = make(map[gethCommon.Address]*big.Int)
		}
		balancesByContracts[model.ETHAddress][from] = new(big.Int)
	}

	// Get balances
	err = s.balancer.BalanceOf(ctx, s.blockHash, balancesByContracts)
	if err != nil {
		s.logger.Error("Failed to get ERC20 balance with ethclient", "len", len(balancesByContracts), "err", err)
		return err
	}

	// Insert balance and calculate diff to total balances
	addrDiff := make(map[gethCommon.Address]map[gethCommon.Address]*big.Int)
	allAddrs := append(addrs, newAddrs...)
	for contractAddr, addrs := range balancesByContracts {
		// Get last recorded balance for these accounts
		latestBalances, err := s.getLatestBalances(ctx, contractAddr, allAddrs)
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
			err := s.accountStore.InsertAccount(ctx, b)
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
	err = s.subStore.BatchUpdateBlockNumber(ctx, s.blockNumber, newAddrs)
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
				b, err := s.subStore.FindTotalBalance(ctx, s.blockNumber-1, token, sub.Group)
				if common.NotFoundError(err) {
					s.logger.Debug("Total balance cannot be found", "group", sub.Group, "number", s.blockNumber-1, "token", token.Hex())
					b = &model.TotalBalance{
						BlockNumber:  s.blockNumber - 1,
						Token:        token.Bytes(),
						Group:        sub.Group,
						Balance:      "0",
						TxFee:        "0",
						MinerReward:  "0",
						UnclesReward: "0",
					}
					err = nil
				} else if err != nil {
					s.logger.Error("Failed to find total balance", "group", sub.Group, "number", s.blockNumber-1, "token", token.Hex(), "err", err)
					return err
				}

				tb, _ = new(big.Int).SetString(b.Balance, 10)
			}
			totalBalances[sub.Group][token] = new(big.Int).Add(tb, d)

			if token != model.ETHAddress {
				continue
			}

			// Consider tx fees
			if f, ok := feeDiff[addr]; ok {
				if totalFees[sub.Group] == nil {
					totalFees[sub.Group] = new(big.Int).Set(f)
				} else {
					totalFees[sub.Group] = new(big.Int).Add(f, totalFees[sub.Group])
				}
			}
			// Consider miner reward
			if r, ok := minerRewardDiff[addr]; ok {
				if totalMinerReward[sub.Group] == nil {
					totalMinerReward[sub.Group] = new(big.Int).Set(r)
				}
			}
			// Consider uncle reward
			if r, ok := uncleRewardDiff[addr]; ok {
				if totalUncleRewards[sub.Group] == nil {
					totalUncleRewards[sub.Group] = new(big.Int).Set(r)
				} else {
					totalUncleRewards[sub.Group] = new(big.Int).Add(r, totalUncleRewards[sub.Group])
				}
			}
		}
	}

	for group, addrs := range totalBalances {
		for token, d := range addrs {
			tb := &model.TotalBalance{
				Token:        token.Bytes(),
				BlockNumber:  s.blockNumber,
				Group:        group,
				TxFee:        "0",
				MinerReward:  "0",
				UnclesReward: "0",
				Balance:      d.String(),
			}

			if token == model.ETHAddress {
				if f, ok := totalFees[group]; ok {
					tb.TxFee = f.String()
				}
				if r, ok := totalMinerReward[group]; ok {
					tb.MinerReward = r.String()
				}
				if r, ok := totalUncleRewards[group]; ok {
					tb.UnclesReward = r.String()
				}
			}
			err = s.subStore.InsertTotalBalance(ctx, tb)
			if err != nil {
				return
			}
		}
	}
	return nil
}

// Get last recorded balance data for these accounts
func (s *transferProcessor) getLatestBalances(ctx context.Context, contractAddr gethCommon.Address, addrs [][]byte) (map[gethCommon.Address]*model.Account, error) {
	balances, err := s.accountStore.FindLatestAccounts(ctx, contractAddr, addrs)
	if err != nil {
		return nil, err
	}
	lastBalances := make(map[gethCommon.Address]*model.Account)
	for _, acct := range balances {
		lastBalances[gethCommon.BytesToAddress(acct.Address)] = acct
	}
	return lastBalances, nil
}

func (s *transferProcessor) printUnexpectedRewardEvent(e *model.Transfer, rewardDiff map[gethCommon.Address]*big.Int) {
	for addr, value := range rewardDiff {
		s.logger.Error("Previous event diff", "block number", e.BlockNumber, "from", string(e.From), "to", addr.Hex(), "balance", value)
	}

	s.logger.Error("Current event diff", "block number", e.BlockNumber, "from", string(e.From), "to", string(e.To), "balance", e.Value)
}
