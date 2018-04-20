package transaction

import (
	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/model"
)

const (
	TableName = "transactions"
)

type Store interface {
	Insert(data *model.Transaction) error
	Find(filter *model.Transaction) (result []model.Transaction, err error)
}

type store struct {
	db *gorm.DB
}

func NewWithDB(db *gorm.DB) Store {
	return &store{
		db: db.Table(TableName),
	}
}

func (t *store) Insert(data *model.Transaction) error {
	return t.db.Create(data).Error
}

func (t *store) Find(filter *model.Transaction) (result []model.Transaction, err error) {
	err = t.db.Where(filter).Find(&result).Error
	return
}
