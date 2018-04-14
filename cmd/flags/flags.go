// Copyright © 2018 AMIS Technologies
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package flags

const (
	ConfigFileFlag = "config"

	// flag names for server
	Host = "host"
	Port = "port"

	// flag names for ethereum service
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
)
