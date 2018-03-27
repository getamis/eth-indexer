package store

import (
	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/indexer/pb"
	headerStore "github.com/maichain/eth-indexer/store/block_header"
	txStore "github.com/maichain/eth-indexer/store/transaction"
	receiptStore "github.com/maichain/eth-indexer/store/transaction_receipt"
)

type StoreManager interface {
	Upsert(header *pb.BlockHeader, transaction []*pb.Transaction, receipts []*pb.TransactionReceipt) error
}

type storeManager struct {
	db *gorm.DB
}

func NewStoreManager(db *gorm.DB) StoreManager {
	return &storeManager{db: db}
}

func (store *storeManager) Upsert(header *pb.BlockHeader, transactions []*pb.Transaction, receipts []*pb.TransactionReceipt) error {
	dbtx := store.db.Begin()
	headerStore := headerStore.NewWithDB(dbtx)
	txStore := txStore.NewWithDB(dbtx)
	receiptStore := receiptStore.NewWithDB(dbtx)

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

	for _, receipt := range receipts {
		err = receiptStore.Upsert(receipt, &pb.TransactionReceipt{})
		if err != nil {
			dbtx.Rollback()
			return err
		}
	}

	dbtx.Commit()

	return nil
}
