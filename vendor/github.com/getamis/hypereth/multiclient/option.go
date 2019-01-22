// Copyright 2018 AMIS Technologies
// This file is part of the hypereth library.
//
// The hypereth library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The hypereth library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the hypereth library. If not, see <http://www.gnu.org/licenses/>.

package multiclient

import (
	"github.com/getamis/sirius/log"
)

// Option represents a Client option
type Option func(*Client) error

// EthURLs represents static ethclient endpoints.
func EthURLs(urls []string) Option {
	return func(mc *Client) error {
		log.Info("EthClients from static list", "urls", urls)
		for _, url := range urls {
			mc.ClientMap().Set(url, nil)
		}
		return nil
	}
}
