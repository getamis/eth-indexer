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

package reorg

import (
	"context"
	"fmt"
	"time"

	"github.com/getamis/eth-indexer/model"
	. "github.com/getamis/eth-indexer/store/sqldb"
)

//go:generate mockery -name Store
type Store interface {
	Insert(ctx context.Context, data *model.Reorg) error
	// For testing
	List(ctx context.Context) ([]*model.Reorg, error)
}

const (
	insertSQL = "INSERT INTO `reorgs2` (`from`, `from_hash`, `to`, `to_hash`, `created_at`) VALUES (%d, X'%s', %d, X'%s', '%s')"
	listSQL   = "SELECT * FROM `reorgs2`"
)

type store struct {
	db DbOrTx
}

func NewWithDB(db DbOrTx) Store {
	return &store{
		db: db,
	}
}

func (s *store) Insert(ctx context.Context, data *model.Reorg) error {
	nowStr := ToTimeStr(time.Now())
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(insertSQL, data.From, Hex(data.FromHash), data.To, Hex(data.ToHash), nowStr))
	return err
}

func (s *store) List(ctx context.Context) ([]*model.Reorg, error) {
	result := []*model.Reorg{}
	err := s.db.SelectContext(ctx, &result, listSQL)
	if err != nil {
		return nil, err
	}
	return result, nil
}
