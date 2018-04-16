package store

import (
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/service/pb"
	headerStore "github.com/maichain/eth-indexer/store/block_header"
	txStore "github.com/maichain/eth-indexer/store/transaction"
	receiptStore "github.com/maichain/eth-indexer/store/transaction_receipt"
)

//go:generate mockery -name Manager

// Manager is a wrapper interface to insert block, receipt and states quickly
type Manager interface {
	// InsertBlock inserts blocks and receipts in db if the block doesn't exist
	InsertBlock(block *types.Block, receipts []*types.Receipt) error
	// LatestHeader returns a latest header from db
	LatestHeader() (*pb.BlockHeader, error)
}

type manager struct {
	db *gorm.DB
}

// NewManager news a store manager to insert block, receipts and states.
func NewManager(db *gorm.DB) Manager {
	return &manager{db: db}
}

func (m *manager) InsertBlock(block *types.Block, receipts []*types.Receipt) (err error) {
	dbtx := m.db.Begin()
	headerStore := headerStore.NewWithDB(dbtx)
	txStore := txStore.NewWithDB(dbtx)
	receiptStore := receiptStore.NewWithDB(dbtx)

	defer func() {
		if err != nil {
			dbtx.Rollback()
			// If it's a duplicate key error, ignore it
			if DuplicateError(err) {
				err = nil
			}
			return
		}
		err = dbtx.Commit().Error
	}()

	// TODO: how to ensure all data are inserted?
	err = headerStore.Insert(Header(block))
	if err != nil {
		return err
	}

	for _, t := range block.Transactions() {
		tx, err := Transaction(block, t)
		if err != nil {
			return err
		}
		err = txStore.Insert(tx)
		if err != nil {
			return err
		}
	}

	for _, r := range receipts {
		err = receiptStore.Insert(Receipt(r))
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *manager) LatestHeader() (*pb.BlockHeader, error) {
	hs := headerStore.NewWithDB(m.db)
	return hs.Last()
}
