/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"google.golang.org/grpc"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/server/egressselector"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/crdb"
	"k8s.io/apiserver/pkg/storage/storagebackend"
	clientset "k8s.io/client-go/kubernetes"
	coreclient "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/klog/v2"

	"go.etcd.io/etcd/client/pkg/v3/transport"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// DeletePodOrErrorf deletes a pod or fails with a call to t.Errorf.
func DeletePodOrErrorf(t *testing.T, c clientset.Interface, ns, name string) {
	if err := c.CoreV1().Pods(ns).Delete(context.TODO(), name, metav1.DeleteOptions{}); err != nil {
		t.Errorf("unable to delete pod %v: %v", name, err)
	}
}

// Requests to try.  Each one should be forbidden or not forbidden
// depending on the authentication and authorization setup of the API server.
var (
	Code200 = map[int]bool{200: true}
	Code201 = map[int]bool{201: true}
	Code400 = map[int]bool{400: true}
	Code401 = map[int]bool{401: true}
	Code403 = map[int]bool{403: true}
	Code404 = map[int]bool{404: true}
	Code405 = map[int]bool{405: true}
	Code503 = map[int]bool{503: true}
)

// WaitForPodToDisappear polls the API server if the pod has been deleted.
func WaitForPodToDisappear(podClient coreclient.PodInterface, podName string, interval, timeout time.Duration) error {
	return wait.PollImmediate(interval, timeout, func() (bool, error) {
		_, err := podClient.Get(context.TODO(), podName, metav1.GetOptions{})
		if err == nil {
			return false, nil
		}
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
}

// GetEtcdClients returns an initialized  clientv3.Client and clientv3.KV.
func GetEtcdClients(config storagebackend.TransportConfig) (*clientv3.Client, clientv3.KV, error) {
	tlsInfo := transport.TLSInfo{
		CertFile:      config.CertFile,
		KeyFile:       config.KeyFile,
		TrustedCAFile: config.TrustedCAFile,
	}

	tlsConfig, err := tlsInfo.ClientConfig()
	if err != nil {
		return nil, nil, err
	}

	cfg := clientv3.Config{
		Endpoints:   config.ServerList,
		DialTimeout: 20 * time.Second,
		DialOptions: []grpc.DialOption{
			grpc.WithBlock(), // block until the underlying connection is up
		},
		TLS: tlsConfig,
	}

	c, err := clientv3.New(cfg)
	if err != nil {
		return nil, nil, err
	}

	return c, clientv3.NewKV(c), nil
}

// GetCRDBClients returns an initialized pool and higher-level client.
func GetCRDBClients(c storagebackend.TransportConfig) (*pgxpool.Pool, storage.InternalTestClient, error) {
	tlsInfo := transport.TLSInfo{
		CertFile:      c.CertFile,
		KeyFile:       c.KeyFile,
		TrustedCAFile: c.TrustedCAFile,
	}
	tlsConfig, err := tlsInfo.ClientConfig()
	if err != nil {
		return nil, nil, err
	}
	// NOTE: Client relies on nil tlsConfig
	// for non-secure connections, update the implicit variable
	if len(c.CertFile) == 0 && len(c.KeyFile) == 0 && len(c.TrustedCAFile) == 0 {
		tlsConfig = nil
	}
	networkContext := egressselector.CRDB.AsNetworkContext()
	var egressDialer utilnet.DialFunc
	if c.EgressLookup != nil {
		egressDialer, err = c.EgressLookup(networkContext)
		if err != nil {
			return nil, nil, err
		}
	}

	cfg, err := pgxpool.ParseConfig(c.ServerList[0])
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse test connection: %v", err)
	}
	cfg.ConnConfig.TLSConfig = tlsConfig
	if egressDialer != nil {
		cfg.ConnConfig.DialFunc = pgconn.DialFunc(egressDialer)
	}
	cfg.ConnConfig.LogLevel = pgx.LogLevelTrace
	cfg.ConnConfig.Logger = NewLogger()
	pool, err := pgxpool.ConnectConfig(context.Background(), cfg)
	return pool, crdb.NewRawClient(pool), err
}

type Logger struct{}

func NewLogger() *Logger {
	return &Logger{}
}

func (l *Logger) Log(ctx context.Context, level pgx.LogLevel, msg string, data map[string]interface{}) {
	var fields []interface{}
	for k, v := range data {
		fields = append(fields, k)
		fields = append(fields, v)
	}
	fields = append(fields, "PGX_LOG_LEVEL")
	fields = append(fields, level)
	switch level {
	case pgx.LogLevelTrace:
		klog.V(4).InfoS(msg, fields...)
	case pgx.LogLevelDebug:
		klog.V(3).InfoS(msg, fields...)
	case pgx.LogLevelInfo:
		klog.V(2).InfoS(msg, fields...)
	case pgx.LogLevelWarn:
		klog.ErrorS(nil, msg, fields...)
	case pgx.LogLevelError:
		klog.ErrorS(nil, msg, fields...)
	default:
		klog.ErrorS(nil, msg, fields...)
	}
}
