package store

import (
	"os"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/indexer/pb"
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

	data := &pb.Transaction{
		Hash: "Hash",
		From: "From",
	}

	out := &pb.Transaction{}

	err = store.Upsert(data, out)
	assert.NoError(t, err, "shouldn't get error:%v", err)
	assert.NotNil(t, out, "out shouldn't be nil")
	assert.Equal(t, out.Hash, data.Hash, "Hash should be equal, exp:%v, got:%v", data.Hash, out.Hash)
	assert.Equal(t, out.From, data.From, "From should be equal, exp:%v, got:%v", data.From, out.From)

	out = &pb.Transaction{}
	filter := &pb.Transaction{Hash: "Hash"}
	transactions, err := store.Find(filter)

	assert.NoError(t, err, "shouldn't get error:%v", err)
	assert.Len(t, transactions, 1, "shold have 1 transaction")
	assert.Equal(t, transactions[0].Hash, filter.Hash, "Hash should be equal, exp:%v, got:%v", filter.Hash, transactions[0].Hash)

	filter = &pb.Transaction{Hash: "not-exist-hash"}
	transactions, err = store.Find(filter)
	assert.NoError(t, err, "shouldn't get error:%v", err)
	assert.Len(t, transactions, 0, "shold have 0 transaction")
}
