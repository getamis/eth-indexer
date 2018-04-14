package transaction_receipt

import (
	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/service/pb"
)

const (
	TableName = "transaction_receipts"
)

type Store interface {
	Insert(data *pb.TransactionReceipt) error
	Upsert(data, result *pb.TransactionReceipt) error
	Find(filter *pb.TransactionReceipt) (result []*pb.TransactionReceipt, err error)
}

type store struct {
	db *gorm.DB
}

func NewWithDB(db *gorm.DB) Store {
	return &store{
		db: db.Table(TableName),
	}
}

func (r *store) Insert(data *pb.TransactionReceipt) error {
	return r.db.Create(data).Error
}

func (r *store) Upsert(data, result *pb.TransactionReceipt) error {
	filter := pb.TransactionReceipt{TxHash: data.TxHash}
	return r.db.Where(filter).Attrs(data).FirstOrCreate(result).Error
}

func (r *store) Find(filter *pb.TransactionReceipt) (result []*pb.TransactionReceipt, err error) {
	err = r.db.Where(filter).Find(&result).Error
	return
}
