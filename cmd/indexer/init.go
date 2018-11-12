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

package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/getamis/eth-indexer/client"
	"github.com/getamis/eth-indexer/store/sqldb"
	"github.com/getamis/sirius/database"
	"github.com/getamis/sirius/database/mysql"
	"github.com/getamis/sirius/metrics"
	vaultApi "github.com/hashicorp/vault/api"
	"github.com/jmoiron/sqlx"
)

func NewEthConn(url string) (client.EthClient, error) {
	return client.NewClient(url)
}

func NewDatabase() (*sqlx.DB, error) {
	option := []interface{}{
		mysql.Database(dbName),
		mysql.Connector(mysql.DefaultProtocol, dbHost, fmt.Sprintf("%d", dbPort)),
		mysql.UserInfo(dbUser, dbPassword),
	}

	if len(dbCAPath) != 0 {
		rootCertPool := x509.NewCertPool()
		pem, err := ioutil.ReadFile(dbCAPath)
		if err != nil {
			panic(err)
		}
		if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
			panic("Failed to append PEM.")
		}
		option = append(option, mysql.EnableTLS(&tls.Config{RootCAs: rootCertPool}))
	}

	return sqldb.New(dbDriver, database.DriverOption(option...))
}

func MustNewVaultClient() *vaultApi.Client {
	tlsConfig := &vaultApi.TLSConfig{
		CACert:   vaultCAPath,
		Insecure: false,
	}

	config := vaultApi.DefaultConfig()
	config.Address = fmt.Sprintf("https://%s", vaultHost)
	err := config.ConfigureTLS(tlsConfig)
	if err != nil {
		panic(err)
	}

	client, err := vaultApi.NewClient(config)
	if err != nil {
		panic(err)
	}
	return client
}

func MetricsDefaultHttpServer(metricsHost string, metricsPort int) *http.Server {
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/metrics", metrics.DefaultRegistry.ServeHTTP)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", metricsHost, metricsPort),
		Handler: serveMux,
	}

	return httpServer
}
