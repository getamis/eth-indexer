package block_header

import (
	"github.com/jinzhu/gorm"
	"github.com/getamis/eth-indexer/model"
)

const (
	TableName   = "block_headers"
	TableNameTd = "total_difficulty"
)

//go:generate mockery -name Store

type Store interface {
	InsertTd(data *model.TotalDifficulty) error
	Insert(data *model.Header) error
	Delete(from, to int64) (err error)
	FindTd(hash []byte) (result *model.TotalDifficulty, err error)
	FindBlockByNumber(blockNumber int64) (result *model.Header, err error)
	FindBlockByHash(hash []byte) (result *model.Header, err error)
	FindLatestBlock() (result *model.Header, err error)
}

type store struct {
	db   *gorm.DB
	tdDb *gorm.DB
}

func NewWithDB(db *gorm.DB) Store {
	return newCacheMiddleware(newWithDB(db))
}

// newWithDB news a new store, for testing use
func newWithDB(db *gorm.DB) Store {
	return &store{
		db:   db.Table(TableName),
		tdDb: db.Table(TableNameTd),
	}
}

func (t *store) InsertTd(data *model.TotalDifficulty) error {
	return t.tdDb.Create(data).Error
}

func (t *store) Insert(data *model.Header) error {
	return t.db.Create(data).Error
}

func (t *store) Delete(from, to int64) error {
	return t.db.Delete(model.Header{}, "number >= ? AND number <= ?", from, to).Error
}

func (t *store) FindTd(hash []byte) (result *model.TotalDifficulty, err error) {
	result = &model.TotalDifficulty{}
	err = t.tdDb.Where("BINARY hash = ?", hash).Limit(1).Find(result).Error
	return
}

func (t *store) FindBlockByNumber(blockNumber int64) (result *model.Header, err error) {
	result = &model.Header{}
	err = t.db.Where("number = ?", blockNumber).Limit(1).Find(result).Error
	return
}

func (t *store) FindBlockByHash(hash []byte) (result *model.Header, err error) {
	result = &model.Header{}
	err = t.db.Where("BINARY hash = ?", hash).Limit(1).Find(result).Error
	return
}

func (t *store) FindLatestBlock() (result *model.Header, err error) {
	result = &model.Header{}
	err = t.db.Order("number DESC").Limit(1).Find(&result).Error
	return
}
