package store

import (
	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/indexer/pb"
)

const (
	TableName = "block_headers"
)

type Store interface {
	Upsert(data, result *pb.BlockHeader) error
	Find(filter *pb.BlockHeader) (result []*pb.BlockHeader, err error)
}

type HeaderStore struct {
	db *gorm.DB
}

func NewWithDB(db *gorm.DB) Store {
	return &HeaderStore{
		db: db.Table(TableName),
	}
}

func (t *HeaderStore) Upsert(data, result *pb.BlockHeader) error {
	filter := pb.BlockHeader{Number: data.Number}
	return t.db.Where(filter).Attrs(data).FirstOrCreate(result).Error
}

func (t *HeaderStore) Find(filter *pb.BlockHeader) (result []*pb.BlockHeader, err error) {
	err = t.db.Where(filter).Find(&result).Error
	return
}
