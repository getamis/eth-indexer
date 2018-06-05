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

const (
	TableName = "transaction_receipts"
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
		db: db.Table(TableName),
	}
}

func (r *store) Insert(data *model.Receipt) error {
	return r.db.Create(data).Error
}

func (r *store) Delete(from, to int64) (err error) {
	err = r.db.Delete(model.Receipt{}, "block_number >= ? AND block_number <= ?", from, to).Error
	return
}

func (r *store) FindReceipt(hash []byte) (result *model.Receipt, err error) {
	result = &model.Receipt{}
	err = r.db.Where("BINARY tx_hash = ?", hash).Limit(1).Find(result).Error
	return
}
