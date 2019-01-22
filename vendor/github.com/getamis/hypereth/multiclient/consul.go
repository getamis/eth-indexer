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
	"fmt"
	netURL "net/url"

	"github.com/getamis/sirius/log"
	consulAPI "github.com/hashicorp/consul/api"
)

// ConsulDiscovery discovers the dynamic ethclient endpoints through consul server.
// TODO: should watch the change of endpoints
func ConsulDiscovery(rawURL, serviceID, serviceScheme string) Option {
	return func(mc *Client) error {
		client, err := createConsulClient(rawURL)
		if err != nil {
			return err
		}
		urls, err := getEthURLsFromConsul(client, serviceID, serviceScheme)
		if err != nil {
			return err
		}
		log.Info("EthClients from consul", "urls", urls)
		for _, url := range urls {
			mc.ClientMap().Set(url, nil)
		}
		return nil
	}
}

func getEthURLsFromConsul(client *consulAPI.Client, serviceID, serviceScheme string) ([]string, error) {
	list, _, err := client.Catalog().Service(serviceID, "", nil)
	if err != nil {
		log.Error("Failed to get service from consul", "serviceID", serviceID, "err", err)
		return nil, err
	}

	ethURLs := make([]string, len(list))
	for i, srv := range list {
		addr := srv.ServiceAddress
		if addr == "" {
			addr = srv.Address
		}
		ethURLs[i] = fmt.Sprintf("%s://%s:%d", serviceScheme, addr, srv.ServicePort)
	}
	return ethURLs, nil
}

func createConsulClient(rawURL string) (*consulAPI.Client, error) {
	consulURL, err := netURL.Parse(rawURL)
	if err != nil {
		log.Error("Failed to parse consul url", "url", rawURL, "err", err)
		return nil, err
	}

	config := &consulAPI.Config{
		Address: consulURL.Host,
		Scheme:  consulURL.Scheme,
	}

	client, err := consulAPI.NewClient(config)
	if err != nil {
		log.Error("Failed to create consul client", "err", err)
		return nil, err
	}
	return client, nil
}
