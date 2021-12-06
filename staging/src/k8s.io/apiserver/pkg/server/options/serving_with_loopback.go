/*
Copyright 2018 The Kubernetes Authors.

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

package options

import (
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/google/uuid"

	"k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/dynamiccertificates"
	"k8s.io/client-go/rest"
	certutil "k8s.io/client-go/util/cert"
)

type SecureServingOptionsWithLoopback struct {
	*SecureServingOptions

	// LoopbackOptions contains authentication and authorization options for the loopback client
	// configuration.
	LoopbackOptions *LoopbackOptions
}

// LoopbackOptions contains authentication and authorization options for the loopback client
// configuration.
type LoopbackOptions struct {
	// Cert is the self-signed certificate. If unset, will be generated when ApplyTo is
	// called.
	Cert []byte
	// Key is the private key for the self-signed certificate. If unset, will be
	// generated when ApplyTo is called.
	Key []byte
	// Token is the bearer token the loopback client uses to communicate with the server. if
	// unset, will be generated when ApplyTo is called.
	Token string
}

func (o *SecureServingOptions) WithLoopback() *SecureServingOptionsWithLoopback {
	return &SecureServingOptionsWithLoopback{
		SecureServingOptions: o,
	}
}

// ApplyTo fills up serving information in the server configuration.
func (s *SecureServingOptionsWithLoopback) ApplyTo(secureServingInfo **server.SecureServingInfo, loopbackClientConfig **rest.Config) error {
	if s == nil || s.SecureServingOptions == nil || secureServingInfo == nil {
		return nil
	}

	if err := s.SecureServingOptions.ApplyTo(secureServingInfo); err != nil {
		return err
	}

	if *secureServingInfo == nil || loopbackClientConfig == nil {
		return nil
	}

	var (
		certPem, keyPem []byte
		loopbackToken   string
		err             error
	)

	if s.LoopbackOptions != nil {
		if len(s.LoopbackOptions.Cert) == 0 {
			return errors.New("when specifying SecureServingOptionsWithLoopback.LoopbackOptions, Cert is required")
		}
		if len(s.LoopbackOptions.Key) == 0 {
			return errors.New("when specifying SecureServingOptionsWithLoopback.LoopbackOptions, Key is required")
		}
		if s.LoopbackOptions.Token == "" {
			return errors.New("when specifying SecureServingOptionsWithLoopback.LoopbackOptions, Token is required")
		}

		certPem = s.LoopbackOptions.Cert
		keyPem = s.LoopbackOptions.Key
		loopbackToken = s.LoopbackOptions.Token
	} else {
		// create self-signed cert+key with the fake server.LoopbackClientServerNameOverride and
		// let the server return it when the loopback client connects.
		certPem, keyPem, err = s.NewLoopbackClientCert()
		if err != nil {
			return fmt.Errorf("failed to generate self-signed certificate for loopback connection: %v", err)
		}

		loopbackToken = uuid.New().String()
	}

	certProvider, err := dynamiccertificates.NewStaticSNICertKeyContent("self-signed loopback", certPem, keyPem, server.LoopbackClientServerNameOverride)
	if err != nil {
		return fmt.Errorf("failed to generate self-signed certificate for loopback connection: %v", err)
	}

	// Write to the front of SNICerts so that this overrides any other certs with the same name
	(*secureServingInfo).SNICerts = append([]dynamiccertificates.SNICertKeyContentProvider{certProvider}, (*secureServingInfo).SNICerts...)

	secureLoopbackClientConfig, err := (*secureServingInfo).NewLoopbackClientConfig(loopbackToken, certPem)
	switch {
	// if we failed and there's no fallback loopback client config, we need to fail
	case err != nil && *loopbackClientConfig == nil:
		(*secureServingInfo).SNICerts = (*secureServingInfo).SNICerts[1:]
		return err

	// if we failed, but we already have a fallback loopback client config (usually insecure), allow it
	case err != nil && *loopbackClientConfig != nil:

	default:
		*loopbackClientConfig = secureLoopbackClientConfig
	}

	return nil
}

// NewLoopbackClientCert creates a self-signed certificate and key for the loopback client.
func (s *SecureServingOptionsWithLoopback) NewLoopbackClientCert() ([]byte, []byte, error) {
	certPem, keyPem, err := certutil.GenerateSelfSignedCertKey(server.LoopbackClientServerNameOverride, nil, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate self-signed certificate for loopback connection: %v", err)
	}
	return certPem, keyPem, nil
}

// NewLoopbackClientConfig creates a loopback client *rest.Config for the given token and
// certificate. API server composers can use this to create a loopback client configuration before
// the server has started. It is slightly different from SecureServingInfo.NewLoopbackClientConfig
// in that this method does not support a BindPort of 0, and it returns an error in that case.
func (s *SecureServingOptionsWithLoopback) NewLoopbackClientConfig(token string, cert []byte) (*rest.Config, error) {
	if s.BindPort == 0 {
		return nil, errors.New("BindPort must be > 0")
	}

	hostAndPort := s.BindAddress.String() + ":" + strconv.Itoa(s.BindPort)
	host, port, err := server.LoopbackHostPort(hostAndPort)
	if err != nil {
		return nil, err
	}

	c := &rest.Config{
		// Do not limit loopback client QPS.
		QPS:  -1,
		Host: "https://" + net.JoinHostPort(host, port),
		TLSClientConfig: rest.TLSClientConfig{
			CAData:     cert,
			ServerName: server.LoopbackClientServerNameOverride,
		},
		BearerToken: token,
	}

	return c, nil
}
