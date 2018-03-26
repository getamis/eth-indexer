package store_manager

import (
	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/indexer/pb"
	headerStore "github.com/maichain/eth-indexer/store/block_header"
	txStore "github.com/maichain/eth-indexer/store/transaction"
)

type StoreManager interface {
	Upsert(header *pb.BlockHeader, transaction []*pb.Transaction) error
}

type storeManager struct {
	db *gorm.DB
}

func NewStoreManager(db *gorm.DB) StoreManager {
	return &storeManager{db: db}
}

func (store *storeManager) Upsert(header *pb.BlockHeader, transactions []*pb.Transaction) error {
	dbtx := store.db.Begin()
	headerStore := headerStore.NewWithDB(dbtx)
	txStore := txStore.NewWithDB(dbtx)

	err := headerStore.Upsert(header, &pb.BlockHeader{})

	if err != nil {
		dbtx.Rollback()
		return err
	}

	for _, tx := range transactions {
		err = txStore.Upsert(tx, &pb.Transaction{})
		if err != nil {
			dbtx.Rollback()
			return err
		}
	}

	dbtx.Commit()

	return nil
}
