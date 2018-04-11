package store

import (
	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/service/pb"
	headerStore "github.com/maichain/eth-indexer/store/block_header"
	txStore "github.com/maichain/eth-indexer/store/transaction"
	receiptStore "github.com/maichain/eth-indexer/store/transaction_receipt"
)

//go:generate mockery -name StoreManager

type StoreManager interface {
	Upsert(header *pb.BlockHeader, transaction []*pb.Transaction, receipts []*pb.TransactionReceipt) error
	GetLatestHeader() (*pb.BlockHeader, error)
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

func (store *storeManager) GetLatestHeader() (*pb.BlockHeader, error) {
	hs := headerStore.NewWithDB(store.db)
	opt := &headerStore.QueryOption{
		Limit:   1,
		OrderBy: "number",
		Order:   headerStore.ORDER_DESC,
	}
	result, _, err := hs.Query(&pb.BlockHeader{}, opt)

	if len(result) > 0 {
		return result[0], nil
	}
	return nil, err
}
