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

package transaction

import (
	"context"
	"testing"

	"github.com/getamis/eth-indexer/common"
	"github.com/getamis/eth-indexer/model"
	"github.com/getamis/eth-indexer/store/sqldb"
	"github.com/getamis/sirius/test"
	"github.com/jmoiron/sqlx"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func makeTx(blockNum int64, blockHex, txHex string) *model.Transaction {
	return &model.Transaction{
		Hash:        common.HexToBytes(txHex),
		BlockHash:   common.HexToBytes(blockHex),
		From:        common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A892"),
		To:          common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A893"),
		Nonce:       10013,
		GasPrice:    123456789,
		GasLimit:    45000,
		Amount:      "4840283445",
		Payload:     []byte{12, 34},
		BlockNumber: blockNum,
	}
}

var _ = Describe("Transaction Database Test", func() {
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
		_, err := db.Exec("DELETE FROM transactions")
		Expect(err).Should(Succeed())
	})

	It("should insert", func() {
		store := NewWithDB(db)
		blockHex := "0x99bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b"

		data1 := makeTx(32100, blockHex, "0x58bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")

		By("insert new transaction")
		err := store.Insert(ctx, data1)
		Expect(err).Should(Succeed())

		By("failed to insert again")
		err = store.Insert(ctx, data1)
		Expect(err).ShouldNot(BeNil())

		data2 := makeTx(32100, blockHex, "0x68bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		By("insert another new transaction")
		err = store.Insert(ctx, data2)
		Expect(err).Should(Succeed())
	})

	It("deletes transactions at a block number", func() {
		store := NewWithDB(db)
		blockHex1 := "0x88bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b"
		blockHex2 := "0x99bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b"
		blockHex3 := "0x77bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b"
		By("insert three new transactions")
		data1 := makeTx(32100, blockHex1, "0x58bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		data2 := makeTx(42100, blockHex2, "0x68bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		data3 := makeTx(42100, blockHex2, "0x78bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		data4 := makeTx(52100, blockHex3, "0x88bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		data := []*model.Transaction{data1, data2, data3, data4}
		for _, tx := range data {
			err := store.Insert(ctx, tx)
			Expect(err).Should(Succeed())
		}

		err := store.Delete(ctx, 42100, 42100)
		Expect(err).Should(Succeed())

		tx, err := store.FindTransaction(ctx, data1.Hash)
		Expect(err).Should(Succeed())
		Expect(*tx).Should(Equal(*data1))
		tx, err = store.FindTransaction(ctx, data2.Hash)
		Expect(common.NotFoundError(err)).Should(BeTrue())
		tx, err = store.FindTransaction(ctx, data3.Hash)
		Expect(common.NotFoundError(err)).Should(BeTrue())
		tx, err = store.FindTransaction(ctx, data4.Hash)
		Expect(err).Should(Succeed())
		Expect(*tx).Should(Equal(*data4))
	})

	It("should get transaction by hash", func() {
		store := NewWithDB(db)
		blockHex1 := "0x88bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b"
		blockHex2 := "0x99bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b"
		By("insert three new transactions")
		data1 := makeTx(32100, blockHex1, "0x58bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		data2 := makeTx(32100, blockHex1, "0x68bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		data3 := makeTx(42100, blockHex2, "0x78bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		data := []*model.Transaction{data1, data2, data3}
		for _, tx := range data {
			err := store.Insert(ctx, tx)
			Expect(err).Should(Succeed())
		}

		transaction, err := store.FindTransaction(ctx, data1.Hash)
		Expect(err).Should(Succeed())
		Expect(*transaction).Should(Equal(*data1))

		transaction, err = store.FindTransaction(ctx, data2.Hash)
		Expect(err).Should(Succeed())
		Expect(*transaction).Should(Equal(*data2))

		transaction, err = store.FindTransaction(ctx, data3.Hash)
		Expect(err).Should(Succeed())
		Expect(*transaction).Should(Equal(*data3))

		By("find an non-existent transaction")
		transaction, err = store.FindTransaction(ctx, data2.BlockHash)
		Expect(common.NotFoundError(err)).Should(BeTrue())
		Expect(transaction).Should(BeNil())
	})

	It("should get transaction by block hash", func() {
		store := NewWithDB(db)
		blockHex1 := "0x88bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b"
		blockHex2 := "0x99bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b"
		By("insert three new transactions")
		data1 := makeTx(32100, blockHex1, "0x58bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		data2 := makeTx(32100, blockHex1, "0x68bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		data3 := makeTx(42100, blockHex2, "0x78bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		data := []*model.Transaction{data1, data2, data3}
		for _, tx := range data {
			err := store.Insert(ctx, tx)
			Expect(err).Should(Succeed())
		}

		transactions, err := store.FindTransactionsByBlockHash(ctx, data1.BlockHash)
		Expect(err).Should(Succeed())
		Expect(2).Should(Equal(len(transactions)))
		Expect(*transactions[0]).Should(Equal(*data1))
		Expect(*transactions[1]).Should(Equal(*data2))
	})
})

func TestTransaction(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Transaction Test")
}
