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

package subscription

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"

	idxCommon "github.com/getamis/eth-indexer/common"
	"github.com/getamis/eth-indexer/model"
	. "github.com/getamis/eth-indexer/store/sqldb"
)

const (
	ErrCodeDuplicateKey uint16 = 1062
)

//go:generate mockery -name Store
type Store interface {
	BatchInsert(ctx context.Context, subs []*model.Subscription) ([]common.Address, error)
	BatchUpdateBlockNumber(ctx context.Context, blockNumber int64, addrs [][]byte) error
	// FindOldSubscriptions find old subscriptions by addresses
	FindOldSubscriptions(ctx context.Context, addrs [][]byte) (result []*model.Subscription, err error)
	Find(ctx context.Context, blockNumber int64, query *model.QueryParameters) (result []*model.Subscription, total uint64, err error)
	FindByGroup(ctx context.Context, groupID int64, query *model.QueryParameters) (result []*model.Subscription, total uint64, err error)
	ListOldSubscriptions(ctx context.Context, query *model.QueryParameters) (result []*model.Subscription, total uint64, err error)

	// Total balance
	InsertTotalBalance(ctx context.Context, data *model.TotalBalance) error
	FindTotalBalance(ctx context.Context, blockNumber int64, token common.Address, group int64) (result *model.TotalBalance, err error)

	Reset(ctx context.Context, from, to int64) error
}

const (
	insertSQL                 = "INSERT INTO `subscriptions` (`block_number`, `group`, `address`, `created_at`, `updated_at`) VALUES (%d, %d, X'%s', '%s', '%s')"
	batchUpdateBlockNumberSQL = "UPDATE `subscriptions` SET `block_number` = %d WHERE `address` IN (%s)"

	findOldSubscriptionsSQL    = "SELECT * FROM `subscriptions` WHERE `address` IN (%s) AND `block_number` > 0"
	findCntSQL                 = "SELECT COUNT(*) FROM `subscriptions` WHERE `block_number` = %d"
	findLmtSQL                 = "SELECT * FROM `subscriptions` WHERE `block_number` = %d LIMIT %d, %d"
	findByGroupCntSQL          = "SELECT COUNT(*) FROM `subscriptions` WHERE `group` = %d"
	findByGroupLmtSQL          = "SELECT * FROM `subscriptions` WHERE `group` = %d LIMIT %d, %d"
	listOldSubscriptionsCntSQL = "SELECT COUNT(*) FROM `subscriptions` WHERE `block_number` > 0"
	listOldSubscriptionsLmtSQL = "SELECT * FROM `subscriptions` WHERE `block_number` > 0 LIMIT %d, %d"

	insertTotalBalanceSQL = "INSERT INTO `total_balances` (`token`, `block_number`, `group`, `balance`, `tx_fee`, `miner_reward`, `uncles_reward`) VALUES (X'%s', %d, %d, '%s', '%s', '%s', '%s')"
	findTotalBalanceSQL   = "SELECT * FROM `total_balances` WHERE `block_number` <= %d AND `token` = X'%s' AND `group` = %d ORDER BY `block_number` DESC LIMIT 1"
	resetSupscriptionsSQL = "UPDATE `subscriptions` SET `block_number` = 0 WHERE `block_number` >= %d AND `block_number` <= %d"
	resetTotalBalanceSQL  = "DELETE FROM `total_balances` WHERE `block_number` >= %d AND `block_number` <= %d"
)

type store struct {
	db DbOrTx
}

func NewWithDB(db DbOrTx) Store {
	return &store{
		db: db,
	}
}

func (t *store) BatchInsert(ctx context.Context, subs []*model.Subscription) (duplicated []common.Address, err error) {
	dbTx, deferFunc, txErr := NewTx(ctx, t.db)
	if txErr != nil {
		return nil, txErr
	}
	defer deferFunc(&err)
	nowStr := ToTimeStr(time.Now())
	for _, sub := range subs {
		_, createErr := dbTx.ExecContext(ctx, fmt.Sprintf(insertSQL, sub.BlockNumber, sub.Group, Hex(sub.Address), nowStr, nowStr))
		if createErr != nil {
			if idxCommon.DuplicateError(createErr) {
				duplicated = append(duplicated, common.BytesToAddress(sub.Address))
			} else {
				return nil, createErr
			}
		}
	}
	return duplicated, nil
}

func (t *store) BatchUpdateBlockNumber(ctx context.Context, blockNumber int64, addrs [][]byte) error {
	if len(addrs) == 0 {
		return nil
	}
	_, err := t.db.ExecContext(ctx, fmt.Sprintf(batchUpdateBlockNumberSQL, blockNumber, InClauseForBytes(addrs)))
	return err
}

func (t *store) FindOldSubscriptions(ctx context.Context, addrs [][]byte) ([]*model.Subscription, error) {
	if len(addrs) == 0 {
		return []*model.Subscription{}, nil
	}

	result := []*model.Subscription{}
	err := t.db.SelectContext(ctx, &result, fmt.Sprintf(findOldSubscriptionsSQL, InClauseForBytes(addrs)))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (t *store) InsertTotalBalance(ctx context.Context, data *model.TotalBalance) error {
	_, err := t.db.ExecContext(ctx, fmt.Sprintf(insertTotalBalanceSQL, Hex(data.Token), data.BlockNumber, data.Group, data.Balance, data.TxFee, data.MinerReward, data.UnclesReward))
	return err
}

func (t *store) FindTotalBalance(ctx context.Context, blockNumber int64, token common.Address, group int64) (*model.TotalBalance, error) {
	result := &model.TotalBalance{}
	err := t.db.GetContext(ctx, result, fmt.Sprintf(findTotalBalanceSQL, blockNumber, Hex(token.Bytes()), group))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (t *store) Reset(ctx context.Context, from, to int64) (err error) {
	// Ensure we are in a db transaction
	dbTx, deferFunc, txErr := NewTx(ctx, t.db)
	if txErr != nil {
		return txErr
	}
	defer deferFunc(&err)

	// Set the block number of subscription to 0
	_, err = dbTx.ExecContext(ctx, fmt.Sprintf(resetSupscriptionsSQL, from, to))
	if err != nil {
		return err
	}
	_, err = dbTx.ExecContext(ctx, fmt.Sprintf(resetTotalBalanceSQL, from, to))
	return err
}

func (t *store) Find(ctx context.Context, blockNumber int64, params *model.QueryParameters) (result []*model.Subscription, total uint64, err error) {
	if params.Page <= 0 {
		return nil, 0, ErrInvalidPage
	}
	if params.Limit <= 0 {
		return nil, 0, ErrInvalidLimit
	}

	err = t.db.GetContext(ctx, &total, fmt.Sprintf(findCntSQL, blockNumber))
	if err != nil {
		return nil, 0, err
	}
	offset := (params.Page - 1) * params.Limit
	err = t.db.SelectContext(ctx, &result, fmt.Sprintf(findLmtSQL, blockNumber, offset, params.Limit))
	if err != nil {
		return nil, 0, err
	}
	return result, total, nil
}

func (t *store) FindByGroup(ctx context.Context, groupID int64, params *model.QueryParameters) (result []*model.Subscription, total uint64, err error) {
	if params.Page <= 0 {
		return nil, 0, ErrInvalidPage
	}
	if params.Limit <= 0 {
		return nil, 0, ErrInvalidLimit
	}

	err = t.db.GetContext(ctx, &total, fmt.Sprintf(findByGroupCntSQL, groupID))
	if err != nil {
		return nil, 0, err
	}
	offset := (params.Page - 1) * params.Limit
	err = t.db.SelectContext(ctx, &result, fmt.Sprintf(findByGroupLmtSQL, groupID, offset, params.Limit))
	if err != nil {
		return nil, 0, err
	}
	return result, total, nil
}

func (t *store) ListOldSubscriptions(ctx context.Context, params *model.QueryParameters) (result []*model.Subscription, total uint64, err error) {
	if params.Page <= 0 {
		return nil, 0, ErrInvalidPage
	}
	if params.Limit <= 0 {
		return nil, 0, ErrInvalidLimit
	}

	err = t.db.GetContext(ctx, &total, fmt.Sprintf(listOldSubscriptionsCntSQL))
	if err != nil {
		return nil, 0, err
	}
	offset := (params.Page - 1) * params.Limit
	err = t.db.SelectContext(ctx, &result, fmt.Sprintf(listOldSubscriptionsLmtSQL, offset, params.Limit))
	if err != nil {
		return nil, 0, err
	}
	return result, total, nil
}
