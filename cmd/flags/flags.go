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

package flags

const (
	ConfigFileFlag = "config"

	// flag names for server
	Host = "host"
	Port = "port"

	// flag names for ethereum service
	EthFlag         = "eth"
	EthProtocolFlag = "eth.protocol"
	EthHostFlag     = "eth.host"
	EthPortFlag     = "eth.port"

	// flag names for database
	DbDriverFlag   = "db.driver"
	DbHostFlag     = "db.host"
	DbPortFlag     = "db.port"
	DbNameFlag     = "db.name"
	DbUserFlag     = "db.user"
	DbPasswordFlag = "db.password"

	// flags for syncing
	SyncTargetBlockFlag      = "sync.targetBlock"
	SyncGetMissingBlocksFlag = "sync.getMissingBlocks"
	SyncFromBlockFlag        = "sync.fromBlock"

	// flags for metrics
	MetricsHostFlag = "metrics.host"
	MetricsPortFlag = "metrics.port"
)
