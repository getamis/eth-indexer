package transaction_receipt

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

	data := model.Receipt{
		CumulativeGasUsed: 43000,
		Bloom:             []byte{12, 34, 66},
		TxHash:            common.HexToBytes("0x58bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b"),
		ContractAddress:   common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A892"),
		GasUsed:           31000,
	}

	err = store.Insert(&data)
	assert.NoError(t, err, "shouldn't get error:%v", err)

	err = store.Insert(&data)
	assert.Error(t, err, "should get duplicate key error")

	filter := model.Receipt{TxHash: data.TxHash}
	transactions, err := store.Find(&filter)

	assert.NoError(t, err, "shouldn't get error:%v", err)
	assert.Len(t, transactions, 1, "should have 1 transaction receipt")
	assert.Equal(t, transactions[0].TxHash, filter.TxHash, "Hash should be equal, exp:%v, got:%v", filter.TxHash, transactions[0].TxHash)

	filter = model.Receipt{TxHash: common.HexToBytes("not-exist-hash")}
	transactions, err = store.Find(&filter)
	assert.NoError(t, err, "shouldn't get error:%v", err)
	assert.Len(t, transactions, 0, "should have 0 transaction receipt")
}
