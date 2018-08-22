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
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/getamis/sirius/log"
)

const (
	ClientGeth   = "Geth"
	ClientParity = "Parity"
)

const (
	ctxTimeout   = 10 * time.Second
	retryTimeout = 3 * time.Second
	dialTimeout  = 5 * time.Second
	fetchCount   = 25
	fetchRound   = 10
	clientType   = ""

	drawField   = "{.DRAW}"
	lengthField = "{.LENGTH}"
	clientField = "{.CLIENT}"
	fetchURL    = "https://www.ethernodes.org/network/1/data?draw={.DRAW}&columns%5B0%5D%5Bdata%5D=id&columns%5B0%5D%5Bname%5D=&columns%5B0%5D%5Bsearchable%5D=true&columns%5B0%5D%5Borderable%5D=true&columns%5B0%5D%5Bsearch%5D%5Bvalue%5D=&columns%5B0%5D%5Bsearch%5D%5Bregex%5D=false&columns%5B1%5D%5Bdata%5D=host&columns%5B1%5D%5Bname%5D=&columns%5B1%5D%5Bsearchable%5D=true&columns%5B1%5D%5Borderable%5D=true&columns%5B1%5D%5Bsearch%5D%5Bvalue%5D=&columns%5B1%5D%5Bsearch%5D%5Bregex%5D=false&columns%5B2%5D%5Bdata%5D=port&columns%5B2%5D%5Bname%5D=&columns%5B2%5D%5Bsearchable%5D=true&columns%5B2%5D%5Borderable%5D=true&columns%5B2%5D%5Bsearch%5D%5Bvalue%5D=&columns%5B2%5D%5Bsearch%5D%5Bregex%5D=false&columns%5B3%5D%5Bdata%5D=country&columns%5B3%5D%5Bname%5D=&columns%5B3%5D%5Bsearchable%5D=true&columns%5B3%5D%5Borderable%5D=true&columns%5B3%5D%5Bsearch%5D%5Bvalue%5D=&columns%5B3%5D%5Bsearch%5D%5Bregex%5D=false&columns%5B4%5D%5Bdata%5D=clientId&columns%5B4%5D%5Bname%5D=&columns%5B4%5D%5Bsearchable%5D=true&columns%5B4%5D%5Borderable%5D=true&columns%5B4%5D%5Bsearch%5D%5Bvalue%5D=&columns%5B4%5D%5Bsearch%5D%5Bregex%5D=false&columns%5B5%5D%5Bdata%5D=client&columns%5B5%5D%5Bname%5D=&columns%5B5%5D%5Bsearchable%5D=true&columns%5B5%5D%5Borderable%5D=true&columns%5B5%5D%5Bsearch%5D%5Bvalue%5D=&columns%5B5%5D%5Bsearch%5D%5Bregex%5D=false&columns%5B6%5D%5Bdata%5D=clientVersion&columns%5B6%5D%5Bname%5D=&columns%5B6%5D%5Bsearchable%5D=true&columns%5B6%5D%5Borderable%5D=true&columns%5B6%5D%5Bsearch%5D%5Bvalue%5D=&columns%5B6%5D%5Bsearch%5D%5Bregex%5D=false&columns%5B7%5D%5Bdata%5D=os&columns%5B7%5D%5Bname%5D=&columns%5B7%5D%5Bsearchable%5D=true&columns%5B7%5D%5Borderable%5D=true&columns%5B7%5D%5Bsearch%5D%5Bvalue%5D=&columns%5B7%5D%5Bsearch%5D%5Bregex%5D=false&columns%5B8%5D%5Bdata%5D=lastUpdate&columns%5B8%5D%5Bname%5D=&columns%5B8%5D%5Bsearchable%5D=true&columns%5B8%5D%5Borderable%5D=true&columns%5B8%5D%5Bsearch%5D%5Bvalue%5D=&columns%5B8%5D%5Bsearch%5D%5Bregex%5D=false&order%5B0%5D%5Bcolumn%5D=8&order%5B0%5D%5Bdir%5D=desc&start=0&length={.LENGTH}&search%5Bvalue%5D={.CLIENT}&search%5Bregex%5D=false"

	gistURL = "https://gist.githubusercontent.com/rfikki/a2ccdc1a31ff24884106da7b9e6a7453/raw/mainnet-peers-latest.txt"
)

var (
	dialer = p2p.TCPDialer{&net.Dialer{Timeout: dialTimeout}}
)

type nodeData struct {
	Nodes []*node `json:"data"`
}

type node struct {
	ID         string    `json:"id"`
	Host       string    `json:"host"`
	Port       int       `json:"port"`
	ClientID   string    `json:"clientId"`
	Client     string    `json:"client"`
	LastUpdate time.Time `json:"lastUpdate"`
}

func ParseNode(nodeURL string) *node {
	u, err := url.Parse(nodeURL)
	if err != nil {
		return nil
	}
	host, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		return nil
	}
	p, _ := strconv.Atoi(port)
	return &node{
		ID:   u.User.String(),
		Host: host,
		Port: p,
	}
}

func (n *node) URL() string {
	return fmt.Sprintf("enode://%s@%s:%d", n.ID, n.Host, n.Port)
}

type PeerMonitor struct {
	ethURL       string
	minPeerCount int
	maxPeerCount int
	quit         chan struct{}
}

func NewPeerMonitor(ethURL string, minPeerCount, maxPeerCount int) *PeerMonitor {
	return &PeerMonitor{
		ethURL:       ethURL,
		minPeerCount: minPeerCount,
		maxPeerCount: maxPeerCount,
		quit:         make(chan struct{}),
	}
}

func (m *PeerMonitor) Run(monitorDuration time.Duration) error {
	// schedule run and force run immediately
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			log.Info("Start to check peer set")
			duration := monitorDuration
			err := m.RunOnce()
			if err != nil {
				log.Error("Failed to check peer set, retry", "err", err)
				duration = retryTimeout
			}
			timer.Stop()
			timer.Reset(duration)
		case <-m.quit:
			return nil
		}
	}
}

func (m *PeerMonitor) Stop() {
	close(m.quit)
}

func (m *PeerMonitor) RunOnce() error {
	dialCtx, dialCancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer dialCancel()
	ethClient, err := Dial(dialCtx, m.ethURL)
	if err != nil {
		return err
	}
	defer ethClient.Close()

	peersCtx, peersCancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer peersCancel()
	peers, err := ethClient.Peers(peersCtx)
	if err != nil {
		return err
	}

	log.Info("Current peers", "count", len(peers))
	if len(peers) > m.minPeerCount {
		log.Info("No need to discover nodes", "minPeerCount", m.minPeerCount)
		return nil
	}

	nodes := fetchNodes(peers, m.maxPeerCount)
	if len(nodes) == 0 {
		log.Error("empty node list")
		return errors.New("empty node list")
	}

	log.Trace("Start to batch add peer")
	addPeerCtx, addPeerCancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer addPeerCancel()
	err = ethClient.BatchAddPeer(addPeerCtx, nodes)
	if err != nil {
		log.Error("Failed to batch add peer", "err", err)
	}

	log.Info("Finish add peer")
	return nil
}

func fetchNodes(curPeers []*p2p.PeerInfo, maxPeerCount int) []string {
	exists := make(map[string]bool)
	for _, p := range curPeers {
		exists[p.ID] = true
	}

	dist := maxPeerCount - len(curPeers)
	enodes := []string{}

	addNodes := func(candidates []*node) {
		for _, c := range candidates {
			exists[c.ID] = true
			enodes = append(enodes, c.URL())
		}
	}

	// fetch from white list in gist
	addNodes(fetchFromWhiteList(exists, gistURL))
	if len(enodes) >= dist {
		enodes = enodes[:dist]
		return enodes
	}

	// fetch nodes from https://www.ethernodes.org/network/1/nodes
	for i := 0; i < fetchRound; i++ {
		addNodes(fetchFromEthNodes(exists, i))
		if len(enodes) >= dist {
			enodes = enodes[:dist]
			break
		}
	}

	return enodes
}

func dialNode(nodeURL string) error {
	ethNode, err := discover.ParseNode(nodeURL)
	if err != nil {
		return err
	}
	conn, err := dialer.Dial(ethNode)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

func fetchFromEthNodes(filter map[string]bool, round int) []*node {
	queryURL := strings.Replace(string(fetchURL), drawField, fmt.Sprintf("%d", round+1), -1)
	queryURL = strings.Replace(string(queryURL), lengthField, fmt.Sprintf("%d", fetchCount), -1)
	queryURL = strings.Replace(string(queryURL), clientField, clientType, -1)

	resp, err := http.Get(queryURL)
	if err != nil {
		log.Error("Failed fetch node data", "url", queryURL, "err", err)
		return nil
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("Failed to read response body", "err", err)
		return nil
	}
	var data nodeData
	err = json.Unmarshal(body, &data)
	if err != nil {
		log.Error("Failed to json unmarshal", "err", err)
		return nil
	}

	isValid := func(n *node) bool {
		if filter[n.ID] {
			return false
		}

		if net.ParseIP(n.Host) == nil {
			return false
		}

		switch n.Client {
		case ClientGeth, ClientParity:
		default:
			return false
		}
		return true
	}

	nodeCh := make(chan *node, len(data.Nodes))
	for _, n := range data.Nodes {
		if !isValid(n) {
			nodeCh <- nil
			continue
		}

		go func(n *node) {
			ports := []int{30303}
			if n.Port != 30303 {
				ports = append(ports, n.Port)
			}

			var err error
			for _, p := range ports {
				n.Port = p
				err = dialNode(n.URL())
				if err == nil {
					break
				}
			}
			if err != nil {
				nodeCh <- nil
			} else {
				nodeCh <- n
			}
		}(n)
	}
	enodes := make([]*node, 0)
	for range data.Nodes {
		if n := <-nodeCh; n != nil {
			enodes = append(enodes, n)
		}
	}
	return enodes
}

var (
	enodeRegExp = regexp.MustCompile("enode:\\/\\/([0-9]|[a-z]|[A-Z])+@[0-9]+(\\.[0-9]+){3}:[0-9]+")
)

func fetchFromWhiteList(filter map[string]bool, listURL string) []*node {
	resp, err := http.Get(listURL)
	if err != nil {
		log.Error("Failed fetch node data", "url", gistURL, "err", err)
		return nil
	}
	defer resp.Body.Close()

	enodes := make([]*node, 0)
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		nodeURLs := enodeRegExp.FindAllString(scanner.Text(), -1)
		for _, nodeURL := range nodeURLs {
			n := ParseNode(nodeURL)
			if !filter[n.ID] {
				enodes = append(enodes, n)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		log.Error("Failed to read response body", "err", err)
		return nil
	}
	return enodes
}
