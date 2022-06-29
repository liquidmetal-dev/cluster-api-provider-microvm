package client

import (
	"context"
	"encoding/base64"
)

type basicAuth struct {
	token           string
	requireSecurity bool
}

// Basic creates a basicAuth with a token.
func Basic(t string, s bool) basicAuth { //nolint: revive // this will not be used
	return basicAuth{token: t, requireSecurity: s}
}

// GetRequestMetadata fullfills the credentials.PerRPCCredentials interface,
// adding the basic auth token to the request authorization header.
func (b basicAuth) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	enc := base64.StdEncoding.EncodeToString([]byte(b.token))

	return map[string]string{
		"authorization": "Basic " + enc,
	}, nil
}

// GetRequestMetadata fullfills the credentials.PerRPCCredentials interface.
func (b basicAuth) RequireTransportSecurity() bool {
	return b.requireSecurity
}
