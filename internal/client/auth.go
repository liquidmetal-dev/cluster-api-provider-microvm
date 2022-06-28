package client

import (
	"context"
	"encoding/base64"
)

type basicAuth struct {
	token string
}

// Basic creates a basicAuth with a token.
func Basic(t string) basicAuth { //nolint: revive // this will not be used
	return basicAuth{token: t}
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
func (basicAuth) RequireTransportSecurity() bool {
	return true
}
