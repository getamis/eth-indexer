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

package uncle_header

import (
	"github.com/getamis/eth-indexer/model"
	"github.com/jinzhu/gorm"
)

//go:generate mockery -name Store

type Store interface {
	Insert(data *model.UncleHeader) error
	DeleteByBlockNumber(from, to int64) (err error)
	FindUncleByHash(hash []byte) (result *model.UncleHeader, err error)
}

type store struct {
	db *gorm.DB
}

func NewWithDB(db *gorm.DB) Store {
	return newCacheMiddleware(newWithDB(db))
}

// newWithDB news a new store, for testing use
func newWithDB(db *gorm.DB) Store {
	return &store{
		db: db,
	}
}

func (t *store) Insert(data *model.UncleHeader) error {
	return t.db.Create(data).Error
}

func (t *store) DeleteByBlockNumber(from, to int64) error {
	return t.db.Delete(model.UncleHeader{}, "block_number >= ? AND block_number <= ?", from, to).Error
}

func (t *store) FindUncleByHash(hash []byte) (result *model.UncleHeader, err error) {
	result = &model.UncleHeader{}
	err = t.db.Where("hash = ?", hash).Limit(1).Find(result).Error
	return
}
