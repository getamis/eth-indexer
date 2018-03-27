package store

import (
	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/indexer/pb"
)

const (
	TableName = "transaction_receipts"
)

type Store interface {
	Upsert(data, result *pb.TransactionReceipt) error
	Find(filter *pb.TransactionReceipt) (result []*pb.TransactionReceipt, err error)
}

type ReceiptStore struct {
	db *gorm.DB
}

func NewWithDB(db *gorm.DB) Store {
	return &ReceiptStore{
		db: db.Table(TableName),
	}
}

func (r *ReceiptStore) Upsert(data, result *pb.TransactionReceipt) error {
	filter := pb.TransactionReceipt{TxHash: data.TxHash}
	return r.db.Where(filter).Attrs(data).FirstOrCreate(result).Error
}

func (r *ReceiptStore) Find(filter *pb.TransactionReceipt) (result []*pb.TransactionReceipt, err error) {
	err = r.db.Where(filter).Find(&result).Error
	return
}
