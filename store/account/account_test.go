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
	"context"
	"fmt"
	"reflect"
	"strconv"
	"testing"

	gethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/getamis/eth-indexer/common"
	"github.com/getamis/eth-indexer/model"
	"github.com/getamis/eth-indexer/store/sqldb"
	"github.com/getamis/sirius/test"
	"github.com/jmoiron/sqlx"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func makeERC20(hexAddr string) *model.ERC20 {
	return &model.ERC20{
		Address: common.HexToBytes(hexAddr),
	}
}

func makeAccount(contractAddress []byte, blockNum int64, hexAddr string) *model.Account {
	return makeAccountWithBalance(contractAddress, blockNum, hexAddr, "987654321098765432109876543210")
}

func makeAccountWithBalance(contractAddress []byte, blockNum int64, hexAddr, balance string) *model.Account {
	return &model.Account{
		ContractAddress: contractAddress,
		BlockNumber:     blockNum,
		Address:         common.HexToBytes(hexAddr),
		Balance:         balance,
	}
}

var _ = Describe("Account Database Test", func() {
	var (
		mysql *test.MySQLContainer
		db    *sqlx.DB
		ctx   = context.Background()
	)
	BeforeSuite(func() {
		var err error
		mysql, err = test.SetupMySQL()
		Expect(mysql).ShouldNot(BeNil())
		Expect(err).Should(Succeed())

		err = test.RunMigrationContainer(mysql, test.MigrationOptions{
			ImageRepository: "quay.io/amis/eth-indexer-db-migration",
		})
		Expect(err).Should(Succeed())

		db, err = sqldb.SimpleConnect("mysql", mysql.URL)
		Expect(err).Should(Succeed())
		Expect(db).ShouldNot(BeNil())
	})

	AfterSuite(func() {
		mysql.Stop()
	})

	BeforeEach(func() {
		// Drop erc20 contract table
		store := NewWithDB(db)
		codes, err := store.ListERC20(ctx)
		Expect(err).Should(BeNil())
		for _, code := range codes {
			_, err := db.Exec(fmt.Sprintf("DROP TABLE %s", model.Transfer{
				Address: code.Address,
			}.TableName()))
			Expect(err).Should(BeNil())
			_, err = db.Exec(fmt.Sprintf("DROP TABLE %s", model.Account{
				ContractAddress: code.Address,
			}.TableName()))
			Expect(err).Should(BeNil())
		}

		_, err = db.Exec("DELETE FROM erc20")
		Expect(err).Should(Succeed())
		// Remove ETH accounts & balances
		_, err = db.Exec("DELETE FROM accounts")
		Expect(err).Should(Succeed())
		_, err = db.Exec("DELETE FROM eth_transfer")
		Expect(err).Should(Succeed())
	})

	It("ListOldERC20(), ListNewERC20(), BatchUpdateERC20BlockNumber()", func() {
		store := NewWithDB(db)
		By("empty erc20 old list")
		r, err := store.ListOldERC20(ctx)
		Expect(err).Should(Succeed())
		Expect(len(r)).Should(BeZero())

		By("empty erc20 new list")
		r, err = store.ListNewERC20(ctx)
		Expect(err).Should(Succeed())
		Expect(len(r)).Should(BeZero())

		By("insert a new erc20")
		newTokens := []*model.ERC20{
			{
				Address: []byte("new0"),
			},
			{
				Address: []byte("new1"),
			},
		}
		for _, token := range newTokens {
			err = store.InsertERC20(ctx, token)
			Expect(err).Should(Succeed())
		}

		By("insert an old erc20")
		oldToken := &model.ERC20{
			BlockNumber: 100,
			Address:     []byte("old"),
		}
		err = store.InsertERC20(ctx, oldToken)
		Expect(err).Should(Succeed())

		By("found new erc20s")
		r, err = store.ListNewERC20(ctx)
		Expect(err).Should(Succeed())
		Expect(r).Should(Equal(newTokens))

		By("found one old erc20")
		r, err = store.ListOldERC20(ctx)
		Expect(err).Should(Succeed())
		Expect(len(r)).Should(Equal(1))
		Expect(r[0]).Should(Equal(oldToken))

		By("set erc20 block number")
		for _, token := range newTokens {
			token.BlockNumber = 199
		}
		err = store.BatchUpdateERC20BlockNumber(ctx, 199, [][]byte{
			newTokens[0].Address,
			newTokens[1].Address,
		})
		Expect(err).Should(Succeed())

		By("found no new erc20")
		r, err = store.ListNewERC20(ctx)
		Expect(err).Should(Succeed())
		Expect(len(r)).Should(BeZero())

		By("found three old erc20s")
		r, err = store.ListOldERC20(ctx)
		Expect(err).Should(Succeed())
		Expect(len(r)).Should(Equal(3))
	})

	Context("InsertAccount()", func() {
		It("inserts one new eth record", func() {
			store := NewWithDB(db)

			data := makeAccount(model.ETHBytes, 1000300, "0xA287a379e6caCa6732E50b88D23c290aA990A892")
			err := store.InsertAccount(ctx, data)
			Expect(err).Should(Succeed())

			err = store.InsertAccount(ctx, data)
			Expect(err).ShouldNot(BeNil())
		})
		It("inserts one new erc20 record", func() {
			store := NewWithDB(db)

			// Insert code to create table
			hexAddr := "0xB287a379e6caCa6732E50b88D23c290aA990A892"
			erc20 := makeERC20(hexAddr)
			err := store.InsertERC20(ctx, erc20)
			Expect(err).Should(Succeed())

			data := makeAccount(erc20.Address, 1000300, "0xA287a379e6caCa6732E50b88D23c290aA990A892")
			err = store.InsertAccount(ctx, data)
			Expect(err).Should(Succeed())

			err = store.InsertAccount(ctx, data)
			Expect(err).ShouldNot(BeNil())
		})
	})

	Context("FindAccount()", func() {
		It("finds the right eth record", func() {
			store := NewWithDB(db)

			data1 := makeAccount(model.ETHBytes, 1000300, "0xA287a379e6caCa6732E50b88D23c290aA990A892")
			err := store.InsertAccount(ctx, data1)
			Expect(err).Should(Succeed())

			data2 := makeAccount(model.ETHBytes, 1000310, "0xA287a379e6caCa6732E50b88D23c290aA990A892")
			err = store.InsertAccount(ctx, data2)
			Expect(err).Should(Succeed())

			data3 := makeAccount(model.ETHBytes, 1000314, "0xC487a379e6caCa6732E50b88D23c290aA990A892")
			err = store.InsertAccount(ctx, data3)
			Expect(err).Should(Succeed())

			// should return this account at latest block number
			account, err := store.FindAccount(ctx, model.ETHAddress, gethCommon.BytesToAddress(data1.Address))
			Expect(err).Should(Succeed())
			Expect(reflect.DeepEqual(*account, *data2)).Should(BeTrue())

			account, err = store.FindAccount(ctx, model.ETHAddress, gethCommon.BytesToAddress(data3.Address))
			Expect(err).Should(Succeed())
			Expect(reflect.DeepEqual(*account, *data3)).Should(BeTrue())

			// if block num is specified, return the exact block number, or the highest
			// block number that's less than the queried block number
			account, err = store.FindAccount(ctx, model.ETHAddress, gethCommon.BytesToAddress(data1.Address), 1000309)
			Expect(err).Should(Succeed())
			Expect(reflect.DeepEqual(*account, *data1)).Should(BeTrue())

			account, err = store.FindAccount(ctx, model.ETHAddress, gethCommon.BytesToAddress(data1.Address), 1000310)
			Expect(err).Should(Succeed())
			Expect(reflect.DeepEqual(*account, *data2)).Should(BeTrue())

			// non-existent account address
			account, err = store.FindAccount(ctx, model.ETHAddress, gethCommon.HexToAddress("0xF287a379e6caCa6732E50b88D23c290aA990A892"))
			Expect(common.NotFoundError(err)).Should(BeTrue())
		})

		It("finds the right erc20 record", func() {
			store := NewWithDB(db)

			// Insert code to create table
			hexAddr := "0xB287a379e6caCa6732E50b88D23c290aA990A892"
			addr := gethCommon.HexToAddress(hexAddr)
			erc20 := makeERC20(hexAddr)
			err := store.InsertERC20(ctx, erc20)
			Expect(err).Should(Succeed())

			data1 := makeAccount(erc20.Address, 1000300, "0xA287a379e6caCa6732E50b88D23c290aA990A892")
			err = store.InsertAccount(ctx, data1)
			Expect(err).Should(Succeed())

			data2 := makeAccount(erc20.Address, 1000310, "0xA287a379e6caCa6732E50b88D23c290aA990A892")
			err = store.InsertAccount(ctx, data2)
			Expect(err).Should(Succeed())

			data3 := makeAccount(erc20.Address, 1000314, "0xC487a379e6caCa6732E50b88D23c290aA990A892")
			err = store.InsertAccount(ctx, data3)
			Expect(err).Should(Succeed())

			// should return this account at latest block number
			account, err := store.FindAccount(ctx, addr, gethCommon.BytesToAddress(data1.Address))
			Expect(err).Should(Succeed())
			Expect(reflect.DeepEqual(*account, *data2)).Should(BeTrue())

			account, err = store.FindAccount(ctx, addr, gethCommon.BytesToAddress(data3.Address))
			Expect(err).Should(Succeed())
			Expect(reflect.DeepEqual(*account, *data3)).Should(BeTrue())

			// if block num is specified, return the exact block number, or the highest
			// block number that's less than the queried block number
			account, err = store.FindAccount(ctx, addr, gethCommon.BytesToAddress(data1.Address), 1000309)
			Expect(err).Should(Succeed())
			Expect(reflect.DeepEqual(*account, *data1)).Should(BeTrue())

			account, err = store.FindAccount(ctx, addr, gethCommon.BytesToAddress(data1.Address), 1000310)
			Expect(err).Should(Succeed())
			Expect(reflect.DeepEqual(*account, *data2)).Should(BeTrue())

			// non-existent account address
			account, err = store.FindAccount(ctx, addr, gethCommon.HexToAddress("0xF287a379e6caCa6732E50b88D23c290aA990A892"))
			Expect(common.NotFoundError(err)).Should(BeTrue())
		})
	})

	Context("FindLatestAccounts()", func() {
		It("finds the eth records with highest block numbers", func() {
			store := NewWithDB(db)
			hexAddr0 := "0xF287a379e6caCa6732E50b88D23c290aA990A892" // does not exist in DB
			hexAddr1 := "0xA287a379e6caCa6732E50b88D23c290aA990A892"
			hexAddr2 := "0xC487a379e6caCa6732E50b88D23c290aA990A892"
			hexAddr3 := "0xD487a379e6caCa6732E50b88D23c290aA990A892"

			blockNumber := int64(1000300)
			var expected []*model.Account
			for _, hexAddr := range []string{hexAddr1, hexAddr2, hexAddr3} {
				var acct *model.Account
				for i := 0; i < 3; i++ {
					acct = makeAccountWithBalance(model.ETHBytes, blockNumber+int64(i), hexAddr, strconv.FormatInt(blockNumber, 10))
					err := store.InsertAccount(ctx, acct)
					Expect(err).Should(Succeed())
				}
				// the last one is with the highest block number
				expected = append(expected, acct)
				blockNumber++
			}

			addrs := [][]byte{common.HexToBytes(hexAddr0), common.HexToBytes(hexAddr1), common.HexToBytes(hexAddr2), common.HexToBytes(hexAddr3), common.HexToBytes(hexAddr3)}
			// should return accounts at latest block number
			accounts, err := store.FindLatestAccounts(ctx, model.ETHAddress, addrs)
			Expect(err).Should(Succeed())
			Expect(len(accounts)).Should(Equal(3))
			for i, acct := range accounts {
				acct.ContractAddress = model.ETHBytes
				Expect(acct).Should(Equal(expected[i]))
			}
		})

		It("finds the erc20 records with highest block numbers", func() {
			store := NewWithDB(db)

			// Insert code to create table
			hexAddr := "0xB287a379e6caCa6732E50b88D23c290aA990A892"
			tokenAddr := gethCommon.HexToAddress(hexAddr)
			erc20 := makeERC20(hexAddr)
			err := store.InsertERC20(ctx, erc20)
			Expect(err).Should(Succeed())

			hexAddr0 := "0xF287a379e6caCa6732E50b88D23c290aA990A892" // does not exist in DB
			hexAddr1 := "0xA287a379e6caCa6732E50b88D23c290aA990A892"
			hexAddr2 := "0xC487a379e6caCa6732E50b88D23c290aA990A892"
			hexAddr3 := "0xD487a379e6caCa6732E50b88D23c290aA990A892"

			blockNumber := int64(1000300)
			var expected []*model.Account
			for _, hexAddr := range []string{hexAddr1, hexAddr2, hexAddr3} {
				var acct *model.Account
				for i := 0; i < 3; i++ {
					acct = makeAccountWithBalance(erc20.Address, blockNumber+int64(i), hexAddr, strconv.FormatInt(blockNumber, 10))
					err := store.InsertAccount(ctx, acct)
					Expect(err).Should(Succeed())
				}
				// the last one is with the highest block number
				expected = append(expected, acct)
				blockNumber++
			}
			addrs := [][]byte{common.HexToBytes(hexAddr0), common.HexToBytes(hexAddr1), common.HexToBytes(hexAddr2), common.HexToBytes(hexAddr3), common.HexToBytes(hexAddr3)}
			// should return accounts at latest block number
			accounts, err := store.FindLatestAccounts(ctx, tokenAddr, addrs)
			Expect(err).Should(Succeed())
			Expect(len(accounts)).Should(Equal(3))
			for i, acct := range accounts {
				acct.ContractAddress = erc20.Address
				Expect(acct).Should(Equal(expected[i]))
			}
		})
	})

	Context("DeleteAccounts()", func() {
		It("deletes eth account states from a block number", func() {
			store := NewWithDB(db)

			data1 := makeAccount(model.ETHBytes, 1000300, "0xA287a379e6caCa6732E50b88D23c290aA990A892")
			data2 := makeAccount(model.ETHBytes, 1000313, "0xA287a379e6caCa6732E50b88D23c290aA990A892")
			data3 := makeAccount(model.ETHBytes, 1000315, "0xC487a379e6caCa6732E50b88D23c290aA990A892")
			data4 := makeAccount(model.ETHBytes, 1000333, "0xC487a379e6caCa6732E50b88D23c290aA990A892")
			data := []*model.Account{data1, data2, data3, data4}
			for _, acct := range data {
				err := store.InsertAccount(ctx, acct)
				Expect(err).Should(Succeed())
			}

			// Delete data2 and data3
			err := store.DeleteAccounts(ctx, model.ETHAddress, 1000301, 1000315)
			Expect(err).Should(Succeed())

			// Found data1 and data4
			account, err := store.FindAccount(ctx, model.ETHAddress, gethCommon.BytesToAddress(data1.Address))
			Expect(err).Should(Succeed())
			Expect(account).Should(Equal(data1))
			account, err = store.FindAccount(ctx, model.ETHAddress, gethCommon.BytesToAddress(data4.Address))
			Expect(err).Should(Succeed())
			Expect(account).Should(Equal(data4))

			// Not found data3
			account, err = store.FindAccount(ctx, model.ETHAddress, gethCommon.BytesToAddress(data3.Address), data3.BlockNumber)
			Expect(err).ShouldNot(Succeed())
		})

		It("deletes erc20 account states from a block number", func() {
			store := NewWithDB(db)

			// Insert code to create table
			hexAddr := "0xB287a379e6caCa6732E50b88D23c290aA990A892"
			addr := gethCommon.HexToAddress(hexAddr)
			erc20 := makeERC20(hexAddr)
			err := store.InsertERC20(ctx, erc20)
			Expect(err).Should(Succeed())

			data1 := makeAccount(erc20.Address, 1000300, "0xA287a379e6caCa6732E50b88D23c290aA990A892")
			data2 := makeAccount(erc20.Address, 1000313, "0xA287a379e6caCa6732E50b88D23c290aA990A892")
			data3 := makeAccount(erc20.Address, 1000315, "0xC487a379e6caCa6732E50b88D23c290aA990A892")
			data4 := makeAccount(erc20.Address, 1000333, "0xC487a379e6caCa6732E50b88D23c290aA990A892")
			data := []*model.Account{data1, data2, data3, data4}
			for _, acct := range data {
				err := store.InsertAccount(ctx, acct)
				Expect(err).Should(Succeed())
			}

			// Delete data2 and data3
			err = store.DeleteAccounts(ctx, addr, 1000301, 1000315)
			Expect(err).Should(Succeed())

			// Found data1 and data4
			account, err := store.FindAccount(ctx, addr, gethCommon.BytesToAddress(data1.Address))
			Expect(err).Should(Succeed())
			Expect(account).Should(Equal(data1))
			account, err = store.FindAccount(ctx, addr, gethCommon.BytesToAddress(data4.Address))
			Expect(err).Should(Succeed())
			Expect(account).Should(Equal(data4))
		})
	})

	Context("InsertTransfer(), DeleteTransfer()", func() {
		It("deletes the right eth transfer", func() {
			store := NewWithDB(db)

			addr1 := common.HexToBytes("0xA287a379e6caCa6732E50b88D23c290aA990A892")
			addr2 := common.HexToBytes("0xB487a379e6caCa6732E50b88D23c290aA990A892")
			addr3 := common.HexToBytes("0xC287a379e6caCa6732E50b88D23c290aA990A892")
			addr4 := common.HexToBytes("0xD287a379e6caCa6732E50b88D23c290aA990A892")
			event1 := &model.Transfer{
				Address:     model.ETHBytes,
				BlockNumber: 101,
				TxHash:      common.HexToBytes("0x01"),
				From:        addr1,
				To:          addr2,
				Value:       "1000000",
			}
			err := store.InsertTransfer(ctx, event1)
			Expect(err).Should(Succeed())

			event2 := &model.Transfer{
				Address:     model.ETHBytes,
				BlockNumber: 106,
				TxHash:      common.HexToBytes("0x11"),
				From:        addr2,
				To:          addr3,
				Value:       "1000000",
			}

			err = store.InsertTransfer(ctx, event2)
			Expect(err).Should(Succeed())

			event3 := &model.Transfer{
				Address:     model.ETHBytes,
				BlockNumber: 110,
				TxHash:      common.HexToBytes("0x21"),
				From:        addr4,
				To:          addr3,
				Value:       "1000000",
			}
			err = store.InsertTransfer(ctx, event3)
			Expect(err).Should(Succeed())

			// FindAllTransfers
			events, err := store.FindAllTransfers(ctx, model.ETHAddress, gethCommon.BytesToAddress(addr1))
			Expect(err).Should(Succeed())
			Expect(len(events)).Should(Equal(1))

			events, err = store.FindAllTransfers(ctx, model.ETHAddress, gethCommon.BytesToAddress(addr2))
			Expect(err).Should(Succeed())
			Expect(len(events)).Should(Equal(2))

			events, err = store.FindAllTransfers(ctx, model.ETHAddress, gethCommon.BytesToAddress(addr3))
			Expect(err).Should(Succeed())
			Expect(len(events)).Should(Equal(2))

			events, err = store.FindAllTransfers(ctx, model.ETHAddress, gethCommon.BytesToAddress(addr4))
			Expect(err).Should(Succeed())
			Expect(len(events)).Should(Equal(1))

			// DeleteTransfer
			err = store.DeleteTransfer(ctx, model.ETHAddress, int64(105), int64(110))
			Expect(err).Should(Succeed())
		})

		It("deletes the right erc20 transfer", func() {
			store := NewWithDB(db)

			// Insert code to create table
			hexAddr := "0xB287a379e6caCa6732E50b88D23c290aA990A892"
			addr := gethCommon.HexToAddress(hexAddr)
			erc20 := makeERC20(hexAddr)
			err := store.InsertERC20(ctx, erc20)
			Expect(err).Should(Succeed())

			addr1 := common.HexToBytes("0xA287a379e6caCa6732E50b88D23c290aA990A892")
			addr2 := common.HexToBytes("0xB487a379e6caCa6732E50b88D23c290aA990A892")
			addr3 := common.HexToBytes("0xC287a379e6caCa6732E50b88D23c290aA990A892")
			addr4 := common.HexToBytes("0xD287a379e6caCa6732E50b88D23c290aA990A892")
			event1 := &model.Transfer{
				Address:     erc20.Address,
				BlockNumber: 101,
				TxHash:      common.HexToBytes("0x01"),
				From:        addr1,
				To:          addr2,
				Value:       "1000000",
			}
			err = store.InsertTransfer(ctx, event1)
			Expect(err).Should(Succeed())

			event2 := &model.Transfer{
				Address:     erc20.Address,
				BlockNumber: 106,
				TxHash:      common.HexToBytes("0x11"),
				From:        addr2,
				To:          addr3,
				Value:       "1000000",
			}

			err = store.InsertTransfer(ctx, event2)
			Expect(err).Should(Succeed())

			event3 := &model.Transfer{
				Address:     erc20.Address,
				BlockNumber: 110,
				TxHash:      common.HexToBytes("0x21"),
				From:        addr4,
				To:          addr3,
				Value:       "1000000",
			}
			err = store.InsertTransfer(ctx, event3)
			Expect(err).Should(Succeed())

			// FindAllTransfers
			events, err := store.FindAllTransfers(ctx, addr, gethCommon.BytesToAddress(addr1))
			Expect(err).Should(Succeed())
			Expect(len(events)).Should(Equal(1))

			events, err = store.FindAllTransfers(ctx, addr, gethCommon.BytesToAddress(addr2))
			Expect(err).Should(Succeed())
			Expect(len(events)).Should(Equal(2))

			events, err = store.FindAllTransfers(ctx, addr, gethCommon.BytesToAddress(addr3))
			Expect(err).Should(Succeed())
			Expect(len(events)).Should(Equal(2))

			events, err = store.FindAllTransfers(ctx, addr, gethCommon.BytesToAddress(addr4))
			Expect(err).Should(Succeed())
			Expect(len(events)).Should(Equal(1))

			// DeleteTransfer
			err = store.DeleteTransfer(ctx, addr, int64(105), int64(110))
			Expect(err).Should(Succeed())

			// FindAllTransfers
			events, err = store.FindAllTransfers(ctx, addr, gethCommon.BytesToAddress(addr1))
			Expect(err).Should(Succeed())
			Expect(len(events)).Should(Equal(1))

			events, err = store.FindAllTransfers(ctx, addr, gethCommon.BytesToAddress(addr2))
			Expect(err).Should(Succeed())
			Expect(len(events)).Should(Equal(1))

			events, err = store.FindAllTransfers(ctx, addr, gethCommon.BytesToAddress(addr3))
			Expect(err).Should(Succeed())
			Expect(len(events)).Should(BeZero())

			events, err = store.FindAllTransfers(ctx, addr, gethCommon.BytesToAddress(addr4))
			Expect(err).Should(Succeed())
			Expect(len(events)).Should(BeZero())
		})
	})

	Context("InsertERC20()", func() {
		It("inserts one new record", func() {
			store := NewWithDB(db)
			hexAddr := "0xB287a379e6caCa6732E50b88D23c290aA990A892"
			data := makeERC20(hexAddr)
			err := store.InsertERC20(ctx, data)
			Expect(err).Should(Succeed())

			err = store.InsertERC20(ctx, data)
			Expect(err).ShouldNot(BeNil())

			// Insert another code at different block number should not alter the original block number
			data2 := makeERC20(hexAddr)
			err = store.InsertERC20(ctx, data2)
			Expect(err).ShouldNot(BeNil())

			code, err := store.FindERC20(ctx, gethCommon.BytesToAddress(data.Address))
			Expect(err).Should(Succeed())
			Expect(reflect.DeepEqual(*code, *data)).Should(BeTrue())

			list, err := store.ListNewERC20(ctx)
			Expect(err).Should(Succeed())
			Expect(list).Should(Equal([]*model.ERC20{data}))
		})
	})

	Context("FindERC20()", func() {
		It("finds the right record", func() {
			store := NewWithDB(db)

			data1 := makeERC20("0xB287a379e6caCa6732E50b88D23c290aA990A892")
			err := store.InsertERC20(ctx, data1)
			Expect(err).Should(Succeed())

			data2 := makeERC20("0xC287a379e6caCa6732E50b88D23c290aA990A892")
			err = store.InsertERC20(ctx, data2)
			Expect(err).Should(Succeed())

			code, err := store.FindERC20(ctx, gethCommon.BytesToAddress(data1.Address))
			Expect(err).Should(Succeed())
			Expect(reflect.DeepEqual(*code, *data1)).Should(BeTrue())

			code, err = store.FindERC20(ctx, gethCommon.BytesToAddress(data2.Address))
			Expect(err).Should(Succeed())
			Expect(reflect.DeepEqual(*code, *data2)).Should(BeTrue())

			// non-existent contract address
			code, err = store.FindERC20(ctx, gethCommon.HexToAddress("0xF287a379e6caCa6732E50b88D23c290aA990A892"))
			Expect(common.NotFoundError(err)).Should(BeTrue())
		})
	})
})

func TestAccount(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Account Database Test")
}
