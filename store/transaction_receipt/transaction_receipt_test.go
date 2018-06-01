package transaction_receipt

import (
	"os"
	"testing"

	"github.com/getamis/sirius/test"
	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/common"
	"github.com/maichain/eth-indexer/model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func makeReceipt(blockNumber int64, txHex string) *model.Receipt {
	return &model.Receipt{
		CumulativeGasUsed: 43000,
		Bloom:             []byte{12, 34, 66},
		TxHash:            common.HexToBytes(txHex),
		ContractAddress:   common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A892"),
		GasUsed:           31000,
		BlockNumber:       blockNumber,
	}
}

var _ = Describe("Receipt Database Test", func() {
	var (
		mysql *test.MySQLContainer
		db    *gorm.DB
	)
	BeforeSuite(func() {
		var err error
		mysql, err = test.NewMySQLContainer("quay.io/amis/eth-indexer-db-migration")
		Expect(mysql).ShouldNot(BeNil())
		Expect(err).Should(Succeed())
		Expect(mysql.Start()).Should(Succeed())

		db, err = gorm.Open("mysql", mysql.URL)
		Expect(err).Should(Succeed())
		Expect(db).ShouldNot(BeNil())

		db.LogMode(os.Getenv("ENABLE_DB_LOG_IN_TEST") != "")
	})

	AfterSuite(func() {
		mysql.Stop()
	})

	BeforeEach(func() {
		db.Table(TableName).Delete(&model.Transaction{})
	})

	It("should insert", func() {
		store := NewWithDB(db)

		data1 := makeReceipt(32100, "0x58bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		data2 := makeReceipt(42100, "0x68bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")

		By("insert new receipt")
		err := store.Insert(data1)
		Expect(err).Should(Succeed())

		By("fail to insert the same receipt")
		err = store.Insert(data1)
		Expect(err).ShouldNot(BeNil())

		By("insert another new receipt")
		err = store.Insert(data2)
		Expect(err).Should(Succeed())
	})

	It("should get receipt by hash", func() {
		store := NewWithDB(db)

		data1 := makeReceipt(32100, "0x58bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		data2 := makeReceipt(42100, "0x68bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		err := store.Insert(data1)
		Expect(err).Should(Succeed())
		err = store.Insert(data2)
		Expect(err).Should(Succeed())

		receipt, err := store.FindReceipt(data1.TxHash)
		Expect(err).Should(Succeed())
		Expect(*receipt).Should(Equal(*data1))

		receipt, err = store.FindReceipt(data2.TxHash)
		Expect(err).Should(Succeed())
		Expect(*receipt).Should(Equal(*data2))

		receipt, err = store.FindReceipt(common.HexToBytes("0x78bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b"))
		Expect(common.NotFoundError(err)).Should(BeTrue())
		Expect(*receipt).Should(Equal(model.Receipt{}))
	})

	It("delete from a block number", func() {
		store := NewWithDB(db)

		data1 := makeReceipt(32100, "0x58bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		data2 := makeReceipt(42100, "0x68bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		data3 := makeReceipt(42100, "0x78bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		data4 := makeReceipt(52100, "0x88bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		data := []*model.Receipt{data1, data2, data3, data4}
		for _, receipt := range data {
			err := store.Insert(receipt)
			Expect(err).Should(Succeed())
		}

		err := store.Delete(42100, 42100)
		receipt, err := store.FindReceipt(data1.TxHash)
		Expect(err).Should(Succeed())
		Expect(*receipt).Should(Equal(*data1))

		receipt, err = store.FindReceipt(data2.TxHash)
		Expect(common.NotFoundError(err)).Should(BeTrue())
		receipt, err = store.FindReceipt(data3.TxHash)
		Expect(common.NotFoundError(err)).Should(BeTrue())

		receipt, err = store.FindReceipt(data4.TxHash)
		Expect(err).Should(Succeed())
		Expect(*receipt).Should(Equal(*data4))
	})
})

func TestReceipt(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Receipt Test")
}
