package account

import (
	"os"
	"testing"

	gethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/common"
	"github.com/maichain/eth-indexer/model"
	"github.com/maichain/mapi/base/test"
	"github.com/maichain/mapi/types/reflect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func makeContract(blockNum int64, hexAddr string) *model.Contract {
	return &model.Contract{
		BlockNumber: blockNum,
		Address:     common.HexToBytes(hexAddr),
		Balance:     "987654321098765432109876543210",
		Nonce:       12345,
		Root:        common.HexToBytes("0x86f9a7ccb763958d0f6c01ea89b7a49eb5a3a8aff0f998ff514b97ad1c4e1fd6"),
		Storage:     []byte{11, 23, 45},
	}
}

func makeContractCode(blockNum int64, hexAddr string) *model.ContractCode {
	return &model.ContractCode{
		BlockNumber: blockNum,
		Address:     common.HexToBytes(hexAddr),
		Hash:        common.HexToBytes("0x86f9a7ccb763958d0f6c01ea89b7a49eb5a3a8aff0f998ff514b97ad1c4e1fd6"),
		Code:        "code",
	}
}

func makeAccount(blockNum int64, hexAddr string) *model.Account {
	return &model.Account{
		BlockNumber: blockNum,
		Address:     common.HexToBytes(hexAddr),
		Balance:     "987654321098765432109876543210",
		Nonce:       12345,
	}
}

var _ = Describe("Account Database Test", func() {
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
		db.Table(NameStateBlocks).Delete(&model.Header{})
		db.Table(NameContractCode).Delete(&model.Header{})
		db.Table(NameContracts).Delete(&model.Header{})
		db.Table(NameAccounts).Delete(&model.Header{})
	})

	Context("InsertAccount()", func() {
		It("inserts one new record", func() {
			store := NewWithDB(db)

			data := makeAccount(1000300, "0xA287a379e6caCa6732E50b88D23c290aA990A892")
			err := store.InsertAccount(data)
			Expect(err).Should(Succeed())

			err = store.InsertAccount(data)
			Expect(err).ShouldNot(BeNil())
		})
	})

	Context("FindAccount()", func() {
		It("finds the right record", func() {
			store := NewWithDB(db)

			data1 := makeAccount(1000300, "0xA287a379e6caCa6732E50b88D23c290aA990A892")
			err := store.InsertAccount(data1)
			Expect(err).Should(Succeed())

			data2 := makeAccount(1000310, "0xA287a379e6caCa6732E50b88D23c290aA990A892")
			err = store.InsertAccount(data2)
			Expect(err).Should(Succeed())

			data3 := makeAccount(1000314, "0xC487a379e6caCa6732E50b88D23c290aA990A892")
			err = store.InsertAccount(data3)
			Expect(err).Should(Succeed())

			// should return this account at latest block number
			account, err := store.FindAccount(gethCommon.BytesToAddress(data1.Address))
			Expect(err).Should(Succeed())
			Expect(reflect.DeepEqual(*account, *data2)).Should(BeTrue())

			account, err = store.FindAccount(gethCommon.BytesToAddress(data3.Address))
			Expect(err).Should(Succeed())
			Expect(reflect.DeepEqual(*account, *data3)).Should(BeTrue())

			// if block num is specified, return the exact block number, or the highest
			// block number that's less than the queried block number
			account, err = store.FindAccount(gethCommon.BytesToAddress(data1.Address), 1000309)
			Expect(err).Should(Succeed())
			Expect(reflect.DeepEqual(*account, *data1)).Should(BeTrue())

			account, err = store.FindAccount(gethCommon.BytesToAddress(data1.Address), 1000310)
			Expect(err).Should(Succeed())
			Expect(reflect.DeepEqual(*account, *data2)).Should(BeTrue())

			// non-existent account address
			account, err = store.FindAccount(gethCommon.StringToAddress("0xF287a379e6caCa6732E50b88D23c290aA990A892"))
			Expect(common.NotFoundError(err)).Should(BeTrue())
		})
	})

	Context("InsertContract()", func() {
		It("inserts one new record", func() {
			store := NewWithDB(db)
			data := makeContract(1000300, "0xB287a379e6caCa6732E50b88D23c290aA990A892")

			err := store.InsertContract(data)
			Expect(err).Should(Succeed())

			err = store.InsertContract(data)
			Expect(err).ShouldNot(BeNil())
		})
	})

	Context("FindContract()", func() {
		It("finds the right record", func() {
			store := NewWithDB(db)

			data1 := makeContract(1000300, "0xA287a379e6caCa6732E50b88D23c290aA990A892")
			err := store.InsertContract(data1)
			Expect(err).Should(Succeed())

			data2 := makeContract(1000310, "0xA287a379e6caCa6732E50b88D23c290aA990A892")
			err = store.InsertContract(data2)
			Expect(err).Should(Succeed())

			data3 := makeContract(1000314, "0xC487a379e6caCa6732E50b88D23c290aA990A892")
			err = store.InsertContract(data3)
			Expect(err).Should(Succeed())

			// should return this contract at latest block number
			account, err := store.FindContract(gethCommon.BytesToAddress(data1.Address))
			Expect(err).Should(Succeed())
			Expect(reflect.DeepEqual(*account, *data2)).Should(BeTrue())

			account, err = store.FindContract(gethCommon.BytesToAddress(data3.Address))
			Expect(err).Should(Succeed())
			Expect(reflect.DeepEqual(*account, *data3)).Should(BeTrue())

			// if block num is specified, return the exact block number, or the highest
			// block number that's less than the queried block number
			account, err = store.FindContract(gethCommon.BytesToAddress(data1.Address), 1000309)
			Expect(err).Should(Succeed())
			Expect(reflect.DeepEqual(*account, *data1)).Should(BeTrue())

			account, err = store.FindContract(gethCommon.BytesToAddress(data1.Address), 1000310)
			Expect(err).Should(Succeed())
			Expect(reflect.DeepEqual(*account, *data2)).Should(BeTrue())

			// non-existent contract address
			account, err = store.FindContract(gethCommon.StringToAddress("0xF287a379e6caCa6732E50b88D23c290aA990A892"))
			Expect(common.NotFoundError(err)).Should(BeTrue())
		})
	})

	Context("InsertContractCode()", func() {
		It("inserts one new record", func() {
			store := NewWithDB(db)
			hexAddr := "0xB287a379e6caCa6732E50b88D23c290aA990A892"
			data := makeContractCode(1000300, hexAddr)
			err := store.InsertContractCode(data)
			Expect(err).Should(Succeed())

			err = store.InsertContractCode(data)
			Expect(err).ShouldNot(BeNil())

			// Insert another code at different block number should not alter the original block number
			data2 := makeContractCode(data.BlockNumber+1, hexAddr)
			err = store.InsertContractCode(data2)
			Expect(err).ShouldNot(BeNil())

			code, err := store.FindContractCode(gethCommon.BytesToAddress(data.Address))
			Expect(err).Should(Succeed())
			Expect(reflect.DeepEqual(*code, *data)).Should(BeTrue())
		})
	})

	Context("FindContractCode()", func() {
		It("finds the right record", func() {
			store := NewWithDB(db)

			data1 := makeContractCode(34000, "0xB287a379e6caCa6732E50b88D23c290aA990A892")
			err := store.InsertContractCode(data1)
			Expect(err).Should(Succeed())

			data2 := makeContractCode(34000, "0xC287a379e6caCa6732E50b88D23c290aA990A892")
			err = store.InsertContractCode(data2)
			Expect(err).Should(Succeed())

			code, err := store.FindContractCode(gethCommon.BytesToAddress(data1.Address))
			Expect(err).Should(Succeed())
			Expect(reflect.DeepEqual(*code, *data1)).Should(BeTrue())

			code, err = store.FindContractCode(gethCommon.BytesToAddress(data2.Address))
			Expect(err).Should(Succeed())
			Expect(reflect.DeepEqual(*code, *data2)).Should(BeTrue())

			// non-existent contract address
			code, err = store.FindContractCode(gethCommon.StringToAddress("0xF287a379e6caCa6732E50b88D23c290aA990A892"))
			Expect(common.NotFoundError(err)).Should(BeTrue())
		})
	})

	Context("InsertStateBlock()", func() {
		It("inserts one new record", func() {
			store := NewWithDB(db)

			data := &model.StateBlock{Number: 3001200}
			err := store.InsertStateBlock(data)
			Expect(err).Should(Succeed())

			err = store.InsertStateBlock(data)
			Expect(err).ShouldNot(BeNil())
		})
	})

	Context("LastStateBlock()", func() {
		It("gets the last state block", func() {
			store := NewWithDB(db)

			err := store.InsertStateBlock(&model.StateBlock{Number: 3001200})
			Expect(err).Should(Succeed())
			err = store.InsertStateBlock(&model.StateBlock{Number: 3001205})
			Expect(err).Should(Succeed())
			err = store.InsertStateBlock(&model.StateBlock{Number: 3001210})
			Expect(err).Should(Succeed())

			block, err := store.LastStateBlock()
			Expect(err).Should(Succeed())
			Expect(block.Number).Should(Equal(int64(3001210)))
		})
	})

	Context("FindStateBlock()", func() {
		It("gets the state block", func() {
			store := NewWithDB(db)

			err := store.InsertStateBlock(&model.StateBlock{Number: 3001200})
			Expect(err).Should(Succeed())
			err = store.InsertStateBlock(&model.StateBlock{Number: 3001205})
			Expect(err).Should(Succeed())
			err = store.InsertStateBlock(&model.StateBlock{Number: 3001210})
			Expect(err).Should(Succeed())

			// we have state for this block
			block, err := store.FindStateBlock(3001200)
			Expect(err).Should(Succeed())
			Expect(block.Number).Should(Equal(int64(3001200)))

			// we don't have state at this block, should find the highest block number
			// that's less than the queried block number
			block, err = store.FindStateBlock(3001207)
			Expect(err).Should(Succeed())
			Expect(block.Number).Should(Equal(int64(3001205)))
		})
	})
})

func TestAccount(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Account Database Test")
}
