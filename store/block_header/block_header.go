package blockheader

import (
	"fmt"

	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/indexer/pb"
)

const (
	TableName = "block_headers"
)

type Store interface {
	Upsert(filter, data, result *pb.BlockHeader) error
	FirstOrCreate(filter, data, result *pb.BlockHeader) error
	Query(filter interface{}, queryOpt *QueryOption) (result []*pb.BlockHeader, err error)
}

type HeaderStore struct {
	db *gorm.DB
}

func NewWithDB(db *gorm.DB) Store {
	return &HeaderStore{
		db: db.Table(TableName),
	}
}

// Upsert updates records matched filter condition with given data,
// or creates a new one with given data.
func (t *HeaderStore) Upsert(filter, data, result *pb.BlockHeader) error {
	return t.db.Where(filter).Assign(data).FirstOrCreate(result).Error
}

// FirstOrCreate returns first record matched filter condition,
// or creates a new one with given data.
func (t *HeaderStore) FirstOrCreate(filter, data, result *pb.BlockHeader) error {
	return t.db.Where(filter).Attrs(data).FirstOrCreate(result).Error
}

// Get returns records matched filter condition and query options.
func (t *HeaderStore) Query(filter interface{}, queryOpt *QueryOption) (result []*pb.BlockHeader, err error) {
	var total int64
	offset := queryOpt.Limit * (queryOpt.Page - 1)

	db := t.db
	db = db.Where(filter)
	if len(queryOpt.Since) > 0 {
		db = db.Where(fmt.Sprintf("%s >= ?", "block_timestamp"), queryOpt.Since)
	}
	if len(queryOpt.Until) > 0 {
		db = db.Where(fmt.Sprintf("%s < ?", "block_timestamp"), queryOpt.Until)
	}

	err = db.Count(&total).Error
	if err != nil {
		return
	}

	if queryOpt != nil {
		if orderBy := queryOpt.OrderString(); len(orderBy) > 0 {
			db = db.Order(orderBy)
		}
		if queryOpt.Limit > 0 {
			db = db.Limit(queryOpt.Limit)
		}
		if offset > 0 {
			db = db.Offset(offset)
		}
	}

	err = db.Find(&result).Error
	if err != nil {
		return
	}

	return
}
