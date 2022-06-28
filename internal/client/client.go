package client

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/url"

	flintlockv1 "github.com/weaveworks-liquidmetal/flintlock/api/services/microvm/v1alpha1"
	flgrpc "github.com/weaveworks-liquidmetal/flintlock/client/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	//+kubebuilder:scaffold:imports
	infrav1 "github.com/weaveworks-liquidmetal/cluster-api-provider-microvm/api/v1alpha1"
	"github.com/weaveworks-liquidmetal/cluster-api-provider-microvm/internal/services/microvm"
)

type clientConfig struct {
	basicAuthToken string
	proxy          *infrav1.Proxy
	tls            *infrav1.TLSConfig
}

// Options is a func to add a option to the flintlock host client.
type Options func(*clientConfig)

// WithBasicAuth adds a basic auth token to the client credentials.
func WithBasicAuth(t string) Options {
	return func(c *clientConfig) {
		c.basicAuthToken = t
	}
}

// WithProxy adds a proxy server to the client.
func WithProxy(p *infrav1.Proxy) Options {
	return func(c *clientConfig) {
		c.proxy = p
	}
}

// WithTLS adds TLS credentials to the client.
func WithTLS(t *infrav1.TLSConfig) Options {
	return func(c *clientConfig) {
		c.tls = t
	}
}

// FactoryFunc is a func to create a new flintlock client.
type FactoryFunc func(address string, opts ...Options) (microvm.Client, error)

// NewFlintlockClient returns a connected client to a flintlock host.
func NewFlintlockClient(address string, opts ...Options) (microvm.Client, error) {
	cfg := buildConfig(opts...)
	creds := insecure.NewCredentials()

	if cfg.tls != nil {
		var err error

		creds, err = loadTLS(cfg.tls)
		if err != nil {
			return nil, err
		}
	}

	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
	}

	if cfg.basicAuthToken != "" {
		dialOpts = append(dialOpts, grpc.WithPerRPCCredentials(Basic(cfg.basicAuthToken)))
	}

	if cfg.proxy != nil {
		url, err := url.Parse(cfg.proxy.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("parsing proxy server url %s: %w", cfg.proxy.Endpoint, err)
		}

		dialOpts = append(dialOpts, flgrpc.WithProxy(url))
	}

	conn, err := grpc.Dial(address, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating grpc connection: %w", err)
	}

	flClient := flintlockv1.NewMicroVMClient(conn)

	return flClient, nil
}

func buildConfig(opts ...Options) clientConfig {
	cfg := clientConfig{}

	for _, opt := range opts {
		opt(&cfg)
	}

	return cfg
}

func loadTLS(cfg *infrav1.TLSConfig) (credentials.TransportCredentials, error) {
	certificate, err := tls.LoadX509KeyPair(cfg.Cert, cfg.Key)
	if err != nil {
		return nil, err
	}

	capool := x509.NewCertPool()
	if !capool.AppendCertsFromPEM([]byte(cfg.CACert)) {
		return nil, fmt.Errorf("could not add cert to pool") //nolint: goerr113 // there is no err to wrap
	}

	tlsConfig := &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{certificate},
		RootCAs:      capool,
	}

	return credentials.NewTLS(tlsConfig), nil
}
