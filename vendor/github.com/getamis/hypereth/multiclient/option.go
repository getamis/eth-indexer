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
	"net/url"

	"github.com/getamis/sirius/log"
	consulAPI "github.com/hashicorp/consul/api"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
)

// Option represents a Client option
type Option func(*Client) error

// EthURLs represents static ethclient endpoints.
func EthURLs(ethURLs []string) Option {
	return func(mc *Client) error {
		log.Info("EthClients from static list", "urls", ethURLs)
		for _, url := range ethURLs {
			mc.rpcClientMap.Set(url, nil)
		}
		return nil
	}
}

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
			mc.rpcClientMap.Set(url, nil)
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
	consulURL, err := url.Parse(rawURL)
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

type KubeConfig struct {
	// The file path to KUBE-CONFIG file
	ConfigPath string
	// The url to override the apiserver address in KUBE-CONFIG file
	APIServer string
}

// K8sEndpointsDiscovery discovers the dynamic ethclient endpoints in k8s cluster.
// There are two ways to access k8s cluster:
// 1. `kubeconfig` is nil means will build in-cluster config with service account token assigned to k8s pod.
// 2. `kubeconfig` is given means access k8s cluster with given apiserver address and KUBE-CONFIG file.
// TODO: should watch the change of endpoints
func K8sEndpointsDiscovery(namespace, service, scheme string, kubeconfig *KubeConfig) Option {
	return func(mc *Client) error {
		kubeClient, err := createKubeClient(kubeconfig)
		if err != nil {
			return err
		}
		urls, err := getEthURLsFromK8s(kubeClient, namespace, service, scheme)
		if err != nil {
			return err
		}
		log.Info("EthClients from k8s cluster", "urls", urls)
		for _, url := range urls {
			mc.rpcClientMap.Set(url, nil)
		}
		return nil
	}
}

func getEthURLsFromK8s(kubeClient clientset.Interface, namespace, service, serviceScheme string) ([]string, error) {
	e, err := kubeClient.CoreV1().Endpoints(namespace).Get(service, meta.GetOptions{})
	if err != nil {
		log.Error("Failed to get endpoints", "namespace", namespace, "name", service, "err", err)
		return nil, err
	}

	ethURLs := make([]string, 0)
	for _, s := range e.Subsets {
		for _, addr := range s.Addresses {
			for _, port := range s.Ports {
				ethURLs = append(ethURLs, fmt.Sprintf("%s://%s:%d", serviceScheme, addr.IP, port.Port))
			}
		}
	}
	return ethURLs, nil
}

func createKubeClient(kubeconfig *KubeConfig) (clientset.Interface, error) {
	var apiserver, configPath string
	if kubeconfig != nil {
		apiserver, configPath = kubeconfig.APIServer, kubeconfig.ConfigPath
	}

	config, err := clientcmd.BuildConfigFromFlags(apiserver, configPath)
	if err != nil {
		log.Error("Failed to create k8s config", "apiserver", apiserver, "kubeconfig", kubeconfig, "err", err)
		return nil, err
	}

	config.UserAgent = "hypereth/multiclient"
	config.AcceptContentTypes = "application/vnd.kubernetes.protobuf,application/json"
	config.ContentType = "application/vnd.kubernetes.protobuf"

	kubeClient, err := clientset.NewForConfig(config)
	if err != nil {
		log.Error("Failed to create k8s client", "err", err)
		return nil, err
	}

	// Informers don't seem to do a good job logging error messages when it
	// can't reach the server, making debugging hard. This makes it easier to
	// figure out if apiserver is configured incorrectly.
	log.Trace("Testing communication with k8s api server")
	_, err = kubeClient.Discovery().ServerVersion()
	if err != nil {
		log.Error("Failed to communicate with k8s api server", "err", err)
		return nil, err
	}
	log.Trace("Communication with k8s api server successful")

	return kubeClient, nil
}
