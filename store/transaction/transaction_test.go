package transaction

import (
	"os"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/common"
	"github.com/maichain/eth-indexer/model"
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

	data := model.Transaction{
		Hash:      common.HexToBytes("0x58bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b"),
		BlockHash: common.HexToBytes("0x58bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b"),
		From:      common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A892"),
		Nonce:     10013,
		GasPrice:  "123456789",
		GasLimit:  45000,
		Amount:    "4840283445",
		Payload:   []byte{12, 34},
	}

	err = store.Insert(&data)
	assert.NoError(t, err, "shouldn't get error:%v", err)

	err = store.Insert(&data)
	assert.Error(t, err, "should get duplicate key error")

	filter := model.Transaction{Hash: data.Hash}
	transactions, err := store.Find(&filter)

	assert.NoError(t, err, "shouldn't get error:%v", err)
	assert.Len(t, transactions, 1, "should have 1 transaction")
	assert.Equal(t, transactions[0].Hash, filter.Hash, "Hash should be equal, exp:%v, got:%v", filter.Hash, transactions[0].Hash)

	filter = model.Transaction{Hash: common.HexToBytes("not-exist-hash")}
	transactions, err = store.Find(&filter)
	assert.NoError(t, err, "shouldn't get error:%v", err)
	assert.Len(t, transactions, 0, "should have 0 transaction")
}
