// Copyright 2018 The eth-indexer Authors
// This file is part of the eth-indexer library.
//
// The eth-indexer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The eth-indexer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the eth-indexer library. If not, see <http://www.gnu.org/licenses/>.

package account

import (
	"os"
	"reflect"
	"testing"

	gethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/getamis/eth-indexer/common"
	"github.com/getamis/eth-indexer/model"
	"github.com/getamis/sirius/test"
	"github.com/jinzhu/gorm"
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
		db.Delete(&model.Header{})

		// Drop erc20 contract storage table
		codes := []model.ERC20{}
		db.Find(&codes)
		for _, code := range codes {
			db.DropTable(model.ERC20Storage{
				Address: code.Address,
			})
			db.DropTable(model.ERC20Transfer{
				Address: code.Address,
			})
		}

		db.Delete(&model.ERC20{})
		db.Delete(&model.Account{})
		db.Delete(&model.ETHTransfer{})
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
			account, err = store.FindAccount(gethCommon.HexToAddress("0xF287a379e6caCa6732E50b88D23c290aA990A892"))
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
			code, err = store.FindERC20(gethCommon.HexToAddress("0xF287a379e6caCa6732E50b88D23c290aA990A892"))
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

	Context("InsertERC20Transfer() & DeleteERC20Transfer()", func() {
		It("deletes the right transfer", func() {
			store := NewWithDB(db)

			// Insert code to create table
			hexAddr := "0xB287a379e6caCa6732E50b88D23c290aA990A892"
			addr := gethCommon.HexToAddress(hexAddr)
			data := makeERC20(hexAddr)
			err := store.InsertERC20(data)
			Expect(err).Should(Succeed())

			event1 := &model.ERC20Transfer{
				Address:     addr.Bytes(),
				BlockNumber: 101,
				TxHash:      common.HexToBytes("0x01"),
				From:        common.HexToBytes("0x02"),
				To:          common.HexToBytes("0x03"),
				Value:       "1000000",
			}
			err = store.InsertERC20Transfer(event1)
			Expect(err).Should(Succeed())

			event2 := &model.ERC20Transfer{
				Address:     addr.Bytes(),
				BlockNumber: 106,
				TxHash:      common.HexToBytes("0x11"),
				From:        common.HexToBytes("0x12"),
				To:          common.HexToBytes("0x13"),
				Value:       "1000000",
			}

			err = store.InsertERC20Transfer(event2)
			Expect(err).Should(Succeed())

			event3 := &model.ERC20Transfer{
				Address:     addr.Bytes(),
				BlockNumber: 110,
				TxHash:      common.HexToBytes("0x21"),
				From:        common.HexToBytes("0x22"),
				To:          common.HexToBytes("0x23"),
				Value:       "1000000",
			}
			err = store.InsertERC20Transfer(event3)
			Expect(err).Should(Succeed())

			err = store.DeleteERC20Transfer(addr, int64(105), int64(110))
			Expect(err).Should(Succeed())
		})
	})

	Context("InsertETHTransfer() & DeleteETHTransfer()", func() {
		It("deletes the right transfer", func() {
			store := NewWithDB(db)

			event1 := &model.ETHTransfer{
				BlockNumber: 101,
				TxHash:      common.HexToBytes("0x01"),
				From:        common.HexToBytes("0x02"),
				To:          common.HexToBytes("0x03"),
				Value:       "1000000",
			}
			err := store.InsertETHTransfer(event1)
			Expect(err).Should(Succeed())

			event2 := &model.ETHTransfer{
				BlockNumber: 106,
				TxHash:      common.HexToBytes("0x11"),
				From:        common.HexToBytes("0x12"),
				To:          common.HexToBytes("0x13"),
				Value:       "1000000",
			}

			err = store.InsertETHTransfer(event2)
			Expect(err).Should(Succeed())

			event3 := &model.ETHTransfer{
				BlockNumber: 110,
				TxHash:      common.HexToBytes("0x21"),
				From:        common.HexToBytes("0x22"),
				To:          common.HexToBytes("0x23"),
				Value:       "1000000",
			}
			err = store.InsertETHTransfer(event3)
			Expect(err).Should(Succeed())

			err = store.DeleteETHTransfer(int64(105), int64(110))
			Expect(err).Should(Succeed())
		})
	})
})

func TestAccount(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Account Database Test")
}
