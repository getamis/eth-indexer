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
	// flag names for server
	Host = "host"
	Port = "port"

	// flag names for ethereum service
	Eth         = "eth"
	EthProtocol = "eth.protocol"
	EthHost     = "eth.host"
	EthPort     = "eth.port"

	// flag names for database
	DbDriver   = "db.driver"
	DbHost     = "db.host"
	DbPort     = "db.port"
	DbName     = "db.name"
	DbUser     = "db.user"
	DbPassword = "db.password"

	// flags for syncing
	SyncTargetBlock      = "sync.targetBlock"
	SyncGetMissingBlocks = "sync.getMissingBlocks"
	SyncFromBlock        = "sync.fromBlock"

	// flags for metrics
	MetricsHost = "metrics.host"
	MetricsPort = "metrics.port"

	//flags for pprof
	PprofEnable  = "pprof"
	PprofPort    = "pprof.port"
	PprofAddress = "pprof.address"
)
