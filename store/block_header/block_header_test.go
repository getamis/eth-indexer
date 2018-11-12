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

package block_header

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

func makeHeader(number int64, hashHex string) *model.Header {
	return &model.Header{
		Hash:                  common.HexToBytes(hashHex),
		ParentHash:            common.HexToBytes("0x35b9253b70be351059982e8d6a218146a18ef9b723e560c7efc540629b4e75f2"),
		UncleHash:             common.HexToBytes("0x2d6159f94932bd669c7161e2563ea4cc0fbf848dd59adbed7df3da74072edd50"),
		Coinbase:              common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A892"),
		Root:                  common.HexToBytes("0x86f9a7ccb763958d0f6c01ea89b7a49eb5a3a8aff0f998ff514b97ad1c4e1fd6"),
		TxHash:                common.HexToBytes("0x3f28c6504aa57084da641571cd710e092c716979dac2664f70fc62cd9d792a4b"),
		ReceiptHash:           common.HexToBytes("0xad2ad2d0fca28f18d0d9fedc7ec2ab4b97277546c212f67519314bfb30f56736"),
		Difficulty:            927399944,
		Number:                number,
		GasLimit:              810000,
		GasUsed:               809999,
		Time:                  123456789,
		MixDigest:             []byte{11, 23, 45},
		Nonce:                 []byte{12, 13, 56, 77},
		ExtraData:             []byte("ExtraData"),
		MinerReward:           "MinerReward",
		UnclesInclusionReward: "UnclesInclusionReward",
		TxsFee:                "TxsFee",
		Uncle1Reward:          "Uncle1Reward",
		Uncle1Coinbase:        []byte("Uncle1Coinbase"),
		Uncle1Hash:            []byte("Uncle1Hash"),
		Uncle2Reward:          "Uncle2Reward",
		Uncle2Coinbase:        []byte("Uncle2Coinbase"),
		Uncle2Hash:            []byte("Uncle2Hash"),
	}
}

var _ = Describe("Block Header Database Test", func() {
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
		_, err := db.Exec("DELETE FROM block_headers")
		Expect(err).Should(Succeed())
		_, err = db.Exec("DELETE FROM total_difficulty")
		Expect(err).Should(Succeed())
	})

	It("should be cached store", func() {
		store := NewWithDB(db, Cache())
		_, ok := store.(*cacheMiddleware)
		Expect(ok).Should(BeTrue())
	})

	It("should get header by hash", func() {
		store := NewWithDB(db)

		data1 := makeHeader(1000300, "0x58bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		data2 := makeHeader(1000301, "0x68bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")

		store.Insert(ctx, data1)
		store.Insert(ctx, data2)

		result, err := store.FindBlockByHash(ctx, data1.Hash)
		Expect(err).Should(Succeed())
		result.CreatedAt = data1.CreatedAt
		result.ID = data1.ID
		Expect(*result).Should(Equal(*data1))

		result, err = store.FindBlockByHash(ctx, data2.Hash)
		Expect(err).Should(Succeed())
		result.CreatedAt = data2.CreatedAt
		result.ID = data2.ID
		Expect(*result).Should(Equal(*data2))

		lastResult, err := store.FindLatestBlock(ctx)
		Expect(err).Should(Succeed())
		lastResult.CreatedAt = data2.CreatedAt
		lastResult.ID = data2.ID
		Expect(*lastResult).Should(Equal(*data2))
	})

	It("should get header by number", func() {
		store := NewWithDB(db)

		data1 := makeHeader(1000300, "0x58bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		data2 := makeHeader(1000301, "0x68bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")

		store.Insert(ctx, data1)
		store.Insert(ctx, data2)

		result, err := store.FindBlockByNumber(ctx, data1.Number)
		Expect(err).Should(Succeed())
		result.CreatedAt = data1.CreatedAt
		result.ID = data1.ID
		Expect(*result).Should(Equal(*data1))

		result, err = store.FindBlockByNumber(ctx, data2.Number)
		Expect(err).Should(Succeed())
		result.CreatedAt = data2.CreatedAt
		result.ID = data2.ID
		Expect(*result).Should(Equal(*data2))
	})

	It("should insert one new record in database", func() {
		By("insert new one header")
		store := NewWithDB(db)
		data := makeHeader(1000300, "0x78bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		err := store.Insert(ctx, data)
		Expect(err).Should(Succeed())

		By("failed to insert again")
		err = store.Insert(ctx, data)
		Expect(err).ShouldNot(BeNil())
	})

	It("deletes header from a block number", func() {
		store := NewWithDB(db)
		data1 := makeHeader(1000300, "0x58bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		data2 := makeHeader(1000301, "0x68bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		data3 := makeHeader(1000303, "0x78bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		data := []*model.Header{data1, data2, data3}
		for _, header := range data {
			err := store.Insert(ctx, header)
			Expect(err).Should(Succeed())
		}

		err := store.Delete(ctx, 1000301, 1000302)
		Expect(err).Should(Succeed())

		result, err := store.FindBlockByNumber(ctx, data1.Number)
		Expect(err).Should(Succeed())
		result.CreatedAt = data1.CreatedAt
		result.ID = data1.ID
		Expect(result).Should(Equal(data1))
		_, err = store.FindBlockByNumber(ctx, data2.Number)
		Expect(common.NotFoundError(err)).Should(BeTrue())
		result, err = store.FindBlockByNumber(ctx, data3.Number)
		Expect(err).Should(Succeed())
		result.CreatedAt = data3.CreatedAt
		result.ID = data3.ID
		Expect(result).Should(Equal(data3))
	})

	It("should get the last header", func() {
		store := NewWithDB(db)

		data1 := makeHeader(1000300, "0x58bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		data2 := makeHeader(1000301, "0x68bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")
		data3 := makeHeader(1000303, "0x78bb59babd8fd8299b22acb997832a75d7b6b666579f80cc281764342f2b373b")

		store.Insert(ctx, data1)
		store.Insert(ctx, data2)
		store.Insert(ctx, data3)

		lastResult, err := store.FindLatestBlock(ctx)
		Expect(err).Should(Succeed())
		lastResult.CreatedAt = data3.CreatedAt
		lastResult.ID = data3.ID
		Expect(*lastResult).Should(Equal(*data3))
	})

	It("should get TD", func() {
		store := NewWithDB(db)

		td := &model.TotalDifficulty{
			Block: 1000,
			Hash:  []byte("1234567890"),
			Td:    "10000000",
		}
		err := store.InsertTd(ctx, td)
		Expect(err).Should(BeNil())

		resTD, err := store.FindTd(ctx, td.Hash)
		Expect(err).Should(BeNil())
		Expect(resTD).Should(Equal(td))

		resTD, err = store.FindTd(ctx, []byte("not found"))
		Expect(common.NotFoundError(err)).Should(BeTrue())
	})
})

func TestBlockHeader(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Block Header Database Test")
}
