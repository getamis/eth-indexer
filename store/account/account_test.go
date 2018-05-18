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

func makeERC20(hexAddr string) *model.ERC20 {
	return &model.ERC20{
		Address: common.HexToBytes(hexAddr),
		Code:    []byte("code"),
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

		// Drop erc20 contract storage table
		codes := []model.ERC20{}
		db.Table(NameERC20).Find(&codes)
		for _, code := range codes {
			db.DropTable(model.ERC20Storage{
				Address: code.Address,
			})
		}

		db.Table(NameERC20).Delete(&model.Header{})
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

	Context("DeleteAccounts()", func() {
		It("deletes account states from a block number", func() {
			store := NewWithDB(db)

			data1 := makeAccount(1000300, "0xA287a379e6caCa6732E50b88D23c290aA990A892")
			data2 := makeAccount(1000313, "0xA287a379e6caCa6732E50b88D23c290aA990A892")
			data3 := makeAccount(1000315, "0xC487a379e6caCa6732E50b88D23c290aA990A892")
			data4 := makeAccount(1000333, "0xC487a379e6caCa6732E50b88D23c290aA990A892")
			data := []*model.Account{data1, data2, data3, data4}
			for _, acct := range data {
				err := store.InsertAccount(acct)
				Expect(err).Should(Succeed())
			}

			// Delete data2 and data3
			err := store.DeleteAccounts(1000301, 1000315)
			Expect(err).Should(Succeed())

			// Found data1 and data4
			account, err := store.FindAccount(gethCommon.BytesToAddress(data1.Address))
			Expect(err).Should(Succeed())
			Expect(account).Should(Equal(data1))
			account, err = store.FindAccount(gethCommon.BytesToAddress(data4.Address))
			Expect(err).Should(Succeed())
			Expect(account).Should(Equal(data4))
		})
	})

	Context("InsertERC20()", func() {
		It("inserts one new record", func() {
			store := NewWithDB(db)
			hexAddr := "0xB287a379e6caCa6732E50b88D23c290aA990A892"
			data := makeERC20(hexAddr)
			err := store.InsertERC20(data)
			Expect(err).Should(Succeed())
			Expect(db.HasTable(model.ERC20Storage{
				Address: data.Address,
			})).Should(BeTrue())

			err = store.InsertERC20(data)
			Expect(err).ShouldNot(BeNil())

			// Insert another code at different block number should not alter the original block number
			data2 := makeERC20(hexAddr)
			err = store.InsertERC20(data2)
			Expect(err).ShouldNot(BeNil())

			code, err := store.FindERC20(gethCommon.BytesToAddress(data.Address))
			Expect(err).Should(Succeed())
			Expect(reflect.DeepEqual(*code, *data)).Should(BeTrue())

			list, err := store.ListERC20()
			Expect(err).Should(Succeed())
			Expect(list).Should(Equal([]model.ERC20{*data}))
		})
	})

	Context("FindERC20()", func() {
		It("finds the right record", func() {
			store := NewWithDB(db)

			data1 := makeERC20("0xB287a379e6caCa6732E50b88D23c290aA990A892")
			err := store.InsertERC20(data1)
			Expect(err).Should(Succeed())
			Expect(db.HasTable(model.ERC20Storage{
				Address: data1.Address,
			})).Should(BeTrue())

			data2 := makeERC20("0xC287a379e6caCa6732E50b88D23c290aA990A892")
			err = store.InsertERC20(data2)
			Expect(err).Should(Succeed())
			Expect(db.HasTable(model.ERC20Storage{
				Address: data2.Address,
			})).Should(BeTrue())

			code, err := store.FindERC20(gethCommon.BytesToAddress(data1.Address))
			Expect(err).Should(Succeed())
			Expect(reflect.DeepEqual(*code, *data1)).Should(BeTrue())

			code, err = store.FindERC20(gethCommon.BytesToAddress(data2.Address))
			Expect(err).Should(Succeed())
			Expect(reflect.DeepEqual(*code, *data2)).Should(BeTrue())

			// non-existent contract address
			code, err = store.FindERC20(gethCommon.StringToAddress("0xF287a379e6caCa6732E50b88D23c290aA990A892"))
			Expect(common.NotFoundError(err)).Should(BeTrue())
		})
	})

	Context("FindERC20Storage()", func() {
		It("finds the right storage", func() {
			store := NewWithDB(db)

			// Insert code to create table
			hexAddr := "0xB287a379e6caCa6732E50b88D23c290aA990A892"
			addr := gethCommon.HexToAddress(hexAddr)
			data := makeERC20(hexAddr)
			err := store.InsertERC20(data)
			Expect(err).Should(Succeed())

			storage1 := &model.ERC20Storage{
				Address:     addr.Bytes(),
				BlockNumber: 101,
				Key:         gethCommon.HexToHash("01").Bytes(),
				Value:       gethCommon.HexToHash("02").Bytes(),
			}
			err = store.InsertERC20Storage(storage1)
			Expect(err).Should(Succeed())

			storage2 := &model.ERC20Storage{
				Address:     addr.Bytes(),
				BlockNumber: 102,
				Key:         gethCommon.HexToHash("01").Bytes(),
				Value:       gethCommon.HexToHash("03").Bytes(),
			}
			err = store.InsertERC20Storage(storage2)
			Expect(err).Should(Succeed())

			s, err := store.FindERC20Storage(addr, gethCommon.BytesToHash(storage1.Key), storage1.BlockNumber)
			Expect(err).Should(Succeed())
			Expect(s).Should(Equal(storage1))

			s, err = store.FindERC20Storage(addr, gethCommon.BytesToHash(storage2.Key), storage2.BlockNumber)
			Expect(err).Should(Succeed())
			Expect(s).Should(Equal(storage2))

			num, err := store.LastSyncERC20Storage(addr, int64(1000))
			Expect(err).Should(Succeed())
			Expect(num).Should(Equal(storage2.BlockNumber))
		})
	})

	Context("DeleteERC20Storage()", func() {
		It("deletes the right storage", func() {
			store := NewWithDB(db)

			// Insert code to create table
			hexAddr := "0xB287a379e6caCa6732E50b88D23c290aA990A892"
			addr := gethCommon.HexToAddress(hexAddr)
			data := makeERC20(hexAddr)
			err := store.InsertERC20(data)
			Expect(err).Should(Succeed())

			storage1 := &model.ERC20Storage{
				Address:     addr.Bytes(),
				BlockNumber: 101,
				Key:         gethCommon.HexToHash("01").Bytes(),
				Value:       gethCommon.HexToHash("02").Bytes(),
			}
			err = store.InsertERC20Storage(storage1)
			Expect(err).Should(Succeed())

			storage2 := &model.ERC20Storage{
				Address:     addr.Bytes(),
				BlockNumber: 106,
				Key:         gethCommon.HexToHash("01").Bytes(),
				Value:       gethCommon.HexToHash("03").Bytes(),
			}
			err = store.InsertERC20Storage(storage2)
			Expect(err).Should(Succeed())

			storage3 := &model.ERC20Storage{
				Address:     addr.Bytes(),
				BlockNumber: 110,
				Key:         gethCommon.HexToHash("01").Bytes(),
				Value:       gethCommon.HexToHash("04").Bytes(),
			}
			err = store.InsertERC20Storage(storage3)
			Expect(err).Should(Succeed())

			for _, storage := range []*model.ERC20Storage{storage1, storage2, storage3} {
				s, err := store.FindERC20Storage(addr, gethCommon.BytesToHash(storage.Key), storage.BlockNumber)
				Expect(err).Should(Succeed())
				Expect(s).Should(Equal(storage))
			}

			err = store.DeleteERC20Storage(addr, int64(105), int64(110))
			Expect(err).Should(Succeed())

			s, err := store.FindERC20Storage(addr, gethCommon.BytesToHash(storage1.Key), storage1.BlockNumber)
			Expect(err).Should(Succeed())
			Expect(s).Should(Equal(storage1))
			for _, storage := range []*model.ERC20Storage{storage2, storage3} {
				s, err := store.FindERC20Storage(addr, gethCommon.BytesToHash(storage.Key), storage.BlockNumber)
				Expect(err).Should(Succeed())
				Expect(s).Should(Equal(storage1))
			}
		})
	})
})

func TestAccount(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Account Database Test")
}
