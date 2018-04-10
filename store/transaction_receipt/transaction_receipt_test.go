package store

import (
	"os"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/service/pb"
	"github.com/maichain/mapi/base/test"
	"github.com/stretchr/testify/assert"
)

func TestTransaction(t *testing.T) {
	mysql, err := test.NewMySQLContainer("quay.io/amis/eth-indexer-db-migration")
	assert.NotNil(t, mysql)
	assert.NoError(t, err)
	assert.NoError(t, mysql.Start())
	defer mysql.Stop()

	db, err := gorm.Open("mysql", mysql.URL)
	assert.NoError(t, err, "should be no error")
	assert.NotNil(t, db, "should not be nil")

	db.LogMode(os.Getenv("ENABLE_DB_LOG_IN_TEST") != "")

	store := NewWithDB(db)

	data := &pb.TransactionReceipt{
		TxHash: "TxHash",
	}

	out := &pb.TransactionReceipt{}

	err = store.Upsert(data, out)
	assert.NoError(t, err, "shouldn't get error:%v", err)
	assert.NotNil(t, out, "out shouldn't be nil")
	assert.Equal(t, out.TxHash, data.TxHash, "TxHash should be equal, exp:%v, got:%v", data.TxHash, out.TxHash)

	out = &pb.TransactionReceipt{}
	filter := &pb.TransactionReceipt{TxHash: "TxHash"}
	transactions, err := store.Find(filter)

	assert.NoError(t, err, "shouldn't get error:%v", err)
	assert.Len(t, transactions, 1, "shold have 1 transaction receipt")
	assert.Equal(t, transactions[0].TxHash, filter.TxHash, "Hash should be equal, exp:%v, got:%v", filter.TxHash, transactions[0].TxHash)

	filter = &pb.TransactionReceipt{TxHash: "not-exist-hash"}
	transactions, err = store.Find(filter)
	assert.NoError(t, err, "shouldn't get error:%v", err)
	assert.Len(t, transactions, 0, "shold have 0 transaction receipt")
}
