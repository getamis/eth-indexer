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

package transaction_receipt

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

func makeReceipt(blockNumber int64, txHex string) *model.Receipt {
	return &model.Receipt{
		CumulativeGasUsed: 43000,
		Root:              []byte("root"),
		Bloom:             []byte{12, 34, 66},
		TxHash:            common.HexToBytes(txHex),
		ContractAddress:   common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A892"),
		GasUsed:           31000,
		BlockNumber:       blockNumber,
		Logs: []*model.Log{
			{
				TxHash:          common.HexToBytes(txHex),
				BlockNumber:     blockNumber,
				ContractAddress: common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A892"),
				EventName:       common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A8222"),
				Topic1:          []byte("topic1"),
				Topic2:          []byte("topic2"),
				Topic3:          []byte("topic3"),
				Data:            common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A8223"),
			},
		},
	}
}

var _ = Describe("Receipt Database Test", func() {
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
		_, err := db.Exec("DELETE FROM transaction_receipts")
		Expect(err).Should(Succeed())
		_, err = db.Exec("DELETE FROM receipt_logs")
		Expect(err).Should(Succeed())
	})

	It("should insert", func() {
		store := NewWithDB(db)

		data1 := makeReceipt(32100, "0x58bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		data2 := makeReceipt(42100, "0x68bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")

		By("insert new receipt")
		err := store.Insert(ctx, data1)
		Expect(err).Should(Succeed())

		By("fail to insert the same receipt")
		err = store.Insert(ctx, data1)
		Expect(err).ShouldNot(BeNil())

		By("insert another new receipt")
		err = store.Insert(ctx, data2)
		Expect(err).Should(Succeed())
	})

	It("should get receipt by hash", func() {
		store := NewWithDB(db)

		data1 := makeReceipt(32100, "0x58bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		data2 := makeReceipt(42100, "0x68bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		err := store.Insert(ctx, data1)
		Expect(err).Should(Succeed())
		err = store.Insert(ctx, data2)
		Expect(err).Should(Succeed())

		receipt, err := store.FindReceipt(ctx, data1.TxHash)
		Expect(err).Should(Succeed())
		Expect(*receipt).Should(Equal(*data1))

		receipt, err = store.FindReceipt(ctx, data2.TxHash)
		Expect(err).Should(Succeed())
		Expect(*receipt).Should(Equal(*data2))

		receipt, err = store.FindReceipt(ctx, common.HexToBytes("0x78bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b"))
		Expect(common.NotFoundError(err)).Should(BeTrue())
	})

	It("delete from a block number", func() {
		store := NewWithDB(db)

		data1 := makeReceipt(32100, "0x58bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		data2 := makeReceipt(42100, "0x68bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		data3 := makeReceipt(42100, "0x78bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		data4 := makeReceipt(52100, "0x88bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		data := []*model.Receipt{data1, data2, data3, data4}
		for _, receipt := range data {
			err := store.Insert(ctx, receipt)
			Expect(err).Should(Succeed())
		}

		err := store.Delete(ctx, 42100, 42100)
		receipt, err := store.FindReceipt(ctx, data1.TxHash)
		Expect(err).Should(Succeed())
		Expect(*receipt).Should(Equal(*data1))

		receipt, err = store.FindReceipt(ctx, data2.TxHash)
		Expect(common.NotFoundError(err)).Should(BeTrue())
		receipt, err = store.FindReceipt(ctx, data3.TxHash)
		Expect(common.NotFoundError(err)).Should(BeTrue())

		receipt, err = store.FindReceipt(ctx, data4.TxHash)
		Expect(err).Should(Succeed())
		Expect(*receipt).Should(Equal(*data4))
	})
})

func TestReceipt(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Receipt Test")
}
