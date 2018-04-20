package block_header

import (
	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/model"
)

const (
	TableName = "block_headers"
)

type Store interface {
	Insert(data *model.Header) error
	Find(filter *model.Header) (result []model.Header, err error)
	// Last returns the header with the greatest number
	Last() (result *model.Header, err error)
}

type store struct {
	db *gorm.DB
}

func NewWithDB(db *gorm.DB) Store {
	return &store{
		db: db.Table(TableName),
	}
}

func (t *store) Insert(data *model.Header) error {
	return t.db.Create(data).Error
}

func (t *store) Find(filter *model.Header) (result []model.Header, err error) {
	err = t.db.Where(filter).Find(&result).Error
	return
}

func (t *store) Last() (result *model.Header, err error) {
	result = &model.Header{}
	err = t.db.Order("number DESC").Limit(1).Find(&result).Error
	return
}
