package store

import (
	"os"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/indexer/pb"
	"github.com/maichain/mapi/base/test"
	"github.com/stretchr/testify/assert"
)

func TestUpsert(t *testing.T) {
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
	data := &pb.BlockHeader{
		ParentHash: "ParentHash",
	}
	out := &pb.BlockHeader{}

	err = store.Upsert(data, out)
	assert.NoError(t, err, "shouldn't get error:%v", err)
	assert.NotNil(t, out, "out shouldn't be nil")
	assert.Equal(t, out.ParentHash, data.ParentHash, "ParentHash should be equal, exp:%v, got:%v", data.ParentHash, out.ParentHash)
}
