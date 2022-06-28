package client

import (
	"fmt"
	"net/url"

	flintlockv1 "github.com/weaveworks-liquidmetal/flintlock/api/services/microvm/v1alpha1"
	flgrpc "github.com/weaveworks-liquidmetal/flintlock/client/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	//+kubebuilder:scaffold:imports
	infrav1 "github.com/weaveworks-liquidmetal/cluster-api-provider-microvm/api/v1alpha1"
	"github.com/weaveworks-liquidmetal/cluster-api-provider-microvm/internal/services/microvm"
)

type clientConfig struct {
	basicAuthToken string
	proxy          *infrav1.Proxy
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

// FactoryFunc is a func to create a new flintlock client.
type FactoryFunc func(address string, opts ...Options) (microvm.Client, error)

// NewFlintlockClient returns a connected client to a flintlock host.
func NewFlintlockClient(address string, opts ...Options) (microvm.Client, error) {
	cfg := buildConfig(opts...)

	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
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
