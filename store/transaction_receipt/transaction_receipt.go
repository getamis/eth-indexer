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

package transaction_receipt

import (
	"github.com/getamis/eth-indexer/model"
	"github.com/jinzhu/gorm"
)

//go:generate mockery -name Store
type Store interface {
	Insert(data *model.Receipt) error
	Delete(from, to int64) (err error)
	FindReceipt(hash []byte) (result *model.Receipt, err error)
}

type store struct {
	db *gorm.DB
}

func NewWithDB(db *gorm.DB) Store {
	return &store{
		db: db,
	}
}

func (r *store) Insert(data *model.Receipt) error {
	// TODO: may need db transaction protection,
	// but mysql doesn't support nested db transaction
	// Insert receipt
	if err := r.db.Create(data).Error; err != nil {
		return err
	}
	// Insert logs
	for _, l := range data.Logs {
		if err := r.db.Create(l).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *store) Delete(from, to int64) error {
	// Delete receipt
	err := r.db.Delete(model.Receipt{}, "block_number >= ? AND block_number <= ?", from, to).Error
	if err != nil {
		return err
	}
	// Delete logs
	err = r.db.Delete(model.Log{}, "block_number >= ? AND block_number <= ?", from, to).Error
	if err != nil {
		return err
	}
	return nil
}

func (r *store) FindReceipt(hash []byte) (*model.Receipt, error) {
	// Find receipt
	receipt := &model.Receipt{}
	err := r.db.Where("tx_hash = ?", hash).Limit(1).Find(receipt).Error
	if err != nil {
		return nil, err
	}

	// Find logs
	logs := []*model.Log{}
	err = r.db.Where("tx_hash = ?", hash).Find(&logs).Error
	if err != nil {
		return nil, err
	}
	receipt.Logs = logs
	return receipt, nil
}
