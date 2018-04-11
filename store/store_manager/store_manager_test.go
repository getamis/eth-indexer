package store

import (
	"os"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/service/pb"
	HeaderStore "github.com/maichain/eth-indexer/store/block_header"
	TxStore "github.com/maichain/eth-indexer/store/transaction"
	ReceiptStore "github.com/maichain/eth-indexer/store/transaction_receipt"
	"github.com/maichain/mapi/base/test"
	"github.com/stretchr/testify/assert"
)

func TestAtomicTransaction(t *testing.T) {
	mysql, err := test.NewMySQLContainer("quay.io/amis/eth-indexer-db-migration")
	assert.NotNil(t, mysql)
	assert.NoError(t, err)
	assert.NoError(t, mysql.Start())
	defer mysql.Stop()

	db, err := gorm.Open("mysql", mysql.URL)
	assert.NoError(t, err, "should be no error")
	assert.NotNil(t, db, "should not be nil")

	db.LogMode(os.Getenv("ENABLE_DB_LOG_IN_TEST") != "")

	manager := NewStoreManager(db)
	block := &pb.BlockHeader{
		ParentHash: "ParentHash",
	}
	txs := []*pb.Transaction{
		{
			Hash: "Hash1",
		},
		{
			Hash: "Hash2",
		},
	}

	receipts := []*pb.TransactionReceipt{
		{
			TxHash: "Hash1",
		},
		{
			TxHash: "Hash2",
		},
	}

	err = manager.Upsert(block, txs, receipts)
	assert.NoError(t, err)

	headerStore := HeaderStore.NewWithDB(db)
	txStore := TxStore.NewWithDB(db)
	receiptStore := ReceiptStore.NewWithDB(db)
	blocks, _ := headerStore.Find(&pb.BlockHeader{})
	txs, _ = txStore.Find(&pb.Transaction{})
	receipts, _ = receiptStore.Find(&pb.TransactionReceipt{})
	assert.Len(t, blocks, 1, "Should have 1 blocks")
	assert.Len(t, txs, 2, "Should have 2 transactions")
	assert.Len(t, receipts, 2, "Should have 2 transaction receipts")
}
