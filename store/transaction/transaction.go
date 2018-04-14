package transaction

import (
	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/service/pb"
)

const (
	TableName = "transactions"
)

type Store interface {
	Insert(data *pb.Transaction) error
	Upsert(data, result *pb.Transaction) error
	Find(filter *pb.Transaction) (result []*pb.Transaction, err error)
}

type store struct {
	db *gorm.DB
}

func NewWithDB(db *gorm.DB) Store {
	return &store{
		db: db.Table(TableName),
	}
}

func (t *store) Insert(data *pb.Transaction) error {
	return t.db.Create(data).Error
}

func (t *store) Upsert(data, result *pb.Transaction) error {
	filter := pb.Transaction{Hash: data.Hash}
	return t.db.Where(filter).Attrs(data).FirstOrCreate(result).Error
}

func (t *store) Find(filter *pb.Transaction) (result []*pb.Transaction, err error) {
	err = t.db.Where(filter).Find(&result).Error
	return
}
