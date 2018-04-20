package transaction_receipt

import (
	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/model"
)

const (
	TableName = "transaction_receipts"
)

type Store interface {
	Insert(data *model.Receipt) error
	Find(filter *model.Receipt) (result []model.Receipt, err error)
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

func (r *store) Find(filter *model.Receipt) (result []model.Receipt, err error) {
	err = r.db.Where(filter).Find(&result).Error
	return
}
