package store

import (
	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/indexer/pb"
)

const (
	TableName = "transactions"
)

type Store interface {
	Upsert(data, result *pb.Transaction) error
	Find(filter *pb.Transaction) (result []*pb.Transaction, err error)
}

type TxStore struct {
	db *gorm.DB
}

func NewWithDB(db *gorm.DB) Store {
	return &TxStore{
		db: db.Table(TableName),
	}
}

func (t *TxStore) Upsert(data, result *pb.Transaction) error {
	filter := pb.Transaction{Hash: data.Hash}
	return t.db.Where(filter).Attrs(data).FirstOrCreate(result).Error
}

func (t *TxStore) Find(filter *pb.Transaction) (result []*pb.Transaction, err error) {
	err = t.db.Where(filter).Find(&result).Error
	return
}
