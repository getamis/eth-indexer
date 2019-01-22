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
	"sync"

	"github.com/getamis/sirius/log"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

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
func K8sEndpointsDiscovery(namespace, name, scheme string, kubeconfig *KubeConfig) Option {
	return func(mc *Client) error {
		kubeClient, err := createKubeClient(kubeconfig)
		if err != nil {
			return err
		}
		lw := createEndpointsListWatch(kubeClient, namespace, name)
		store := newEndpointStore(mc.ClientMap(), scheme)
		// Force sync at first
		err = syncWith(&lw, store)
		if err != nil {
			log.Error("Failed to sync the latest endpoints", "err", err)
			return err
		}
		reflector := cache.NewReflector(&lw, &v1.Endpoints{}, store, 0)
		go reflector.Run(mc.Context().Done())
		return nil
	}
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

func createEndpointsListWatch(kubeClient clientset.Interface, ns, name string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			// list with specific name
			opts.FieldSelector = fields.OneTermEqualSelector("metadata.name", name).String()
			return kubeClient.CoreV1().Endpoints(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			// watch with specific name
			opts.FieldSelector = fields.OneTermEqualSelector("metadata.name", name).String()
			return kubeClient.CoreV1().Endpoints(ns).Watch(opts)
		},
	}
}

func syncWith(lw cache.ListerWatcher, store cache.Store) error {
	// Explicitly set "0" as resource version - it's fine for the List()
	// to be served from cache and potentially be delayed relative to
	// etcd contents. Reflector framework will catch up via Watch() eventually.
	options := metav1.ListOptions{ResourceVersion: "0"}
	list, err := lw.List(options)
	if err != nil {
		return err
	}

	listMetaInterface, err := meta.ListAccessor(list)
	if err != nil {
		return err
	}

	resourceVersion := listMetaInterface.GetResourceVersion()

	items, err := meta.ExtractList(list)
	if err != nil {
		return err
	}

	found := make([]interface{}, 0, len(items))
	for _, item := range items {
		found = append(found, item)
	}
	return store.Replace(found, resourceVersion)
}

// endpointStore implements the k8s.io/kubernetes/client-go/tools/cache.Store
// interface. Instead of storing entire Kubernetes objects, it stores urls of ethclients
// generated based on those objects.
type endpointStore struct {
	// Protects metrics
	mutex           sync.RWMutex
	resourceVersion string
	scheme          string
	endpoints       map[types.UID][]string
	rpcClientMap    *Map
}

func newEndpointStore(rpcClientMap *Map, scheme string) *endpointStore {
	return &endpointStore{
		scheme:       scheme,
		endpoints:    map[types.UID][]string{},
		rpcClientMap: rpcClientMap,
	}
}

// Implementing k8s.io/kubernetes/client-go/tools/cache.Store interface

func (s *endpointStore) Add(obj interface{}) error {
	o, err := meta.Accessor(obj)
	if err != nil {
		return err
	}

	endpoint := obj.(*v1.Endpoints)
	urls := getEthURLsFromK8sEndpoint(endpoint, s.scheme)

	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, url := range urls {
		s.rpcClientMap.Set(url, nil)
	}

	s.endpoints[o.GetUID()] = urls

	return nil
}

func (s *endpointStore) Update(obj interface{}) error {
	o, err := meta.Accessor(obj)
	if err != nil {
		return err
	}
	endpoint := obj.(*v1.Endpoints)
	news := getEthURLsFromK8sEndpoint(endpoint, s.scheme)

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Add new urls
	for _, url := range news {
		if c := s.rpcClientMap.Get(url); c == nil {
			s.rpcClientMap.Set(url, nil)
		}
	}

	// Remove old urls
	olds := s.endpoints[o.GetUID()]
	removed := findRemoved(olds, news)
	for _, url := range removed {
		s.rpcClientMap.Delete(url)
	}

	s.endpoints[o.GetUID()] = news

	return nil
}

func (s *endpointStore) Delete(obj interface{}) error {
	o, err := meta.Accessor(obj)
	if err != nil {
		return err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	urls := s.endpoints[o.GetUID()]

	for _, url := range urls {
		s.rpcClientMap.Delete(url)
	}

	delete(s.endpoints, o.GetUID())

	return nil
}

func (s *endpointStore) List() []interface{} {
	return nil
}

func (s *endpointStore) ListKeys() []string {
	return nil
}

func (s *endpointStore) Get(obj interface{}) (item interface{}, exists bool, err error) {
	return nil, false, nil
}

func (s *endpointStore) GetByKey(key string) (item interface{}, exists bool, err error) {
	return nil, false, nil
}

// Replace will delete the contents of the store, using instead the
// given list.
func (s *endpointStore) Replace(list []interface{}, resourceVersion string) error {
	if s.checkResourceVersion(resourceVersion) {
		log.Trace("Resource version is not changed, ignore replace")
		return nil
	}
	s.mutex.Lock()
	for _, urls := range s.endpoints {
		for _, url := range urls {
			s.rpcClientMap.Delete(url)
		}
	}
	s.endpoints = map[types.UID][]string{}
	s.mutex.Unlock()

	for _, o := range list {
		err := s.Add(o)
		if err != nil {
			return err
		}
	}

	s.mutex.Lock()
	s.resourceVersion = resourceVersion
	s.mutex.Unlock()
	return nil
}

func (s *endpointStore) Resync() error {
	return nil
}

func (s *endpointStore) checkResourceVersion(rv string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.resourceVersion == rv
}

func getEthURLsFromK8sEndpoint(e *v1.Endpoints, scheme string) []string {
	ethURLs := make([]string, 0)
	for _, s := range e.Subsets {
		for _, addr := range s.Addresses {
			for _, port := range s.Ports {
				ethURLs = append(ethURLs, fmt.Sprintf("%s://%s:%d", scheme, addr.IP, port.Port))
			}
		}
	}
	return ethURLs
}

// findRemoved returns the string array represents the elements in olds array and removed
// in news array.
func findRemoved(olds, news []string) []string {
	removed := []string{}
	for _, o := range olds {
		find := false
		for _, n := range news {
			if o == n {
				find = true
				break
			}
		}
		if !find {
			removed = append(removed, o)
		}
	}
	return removed
}
