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

package contracts

import (
	"math/big"
	"testing"

	"github.com/getamis/sirius/test"
	"github.com/jinzhu/gorm"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	mysql *test.MySQLContainer
	db    *gorm.DB
)

var _ = Describe("Contracts Test", func() {
	It("DecodeDecimals()", func() {
		d, err := DecodeDecimals([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 18})
		Expect(err).Should(BeNil())
		Expect(d).Should(Equal(uint8(18)))
	})

	It("DecodeBalanceOf()", func() {
		d, err := DecodeBalanceOf([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 13, 224, 182, 179, 167, 100, 0, 0})
		Expect(err).Should(BeNil())
		exp, _ := new(big.Int).SetString("1000000000000000000", 10)
		Expect(d).Should(Equal(exp))
	})
})

func TestContracts(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Contracts Test")
}
