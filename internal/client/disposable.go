package client

import (
	"context"

	flintlockv1 "github.com/weaveworks-liquidmetal/flintlock/api/services/microvm/v1alpha1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type disposableClient struct {
	flintlockClient flintlockv1.MicroVMClient
	conn            *grpc.ClientConn
}

func (c *disposableClient) Dispose() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *disposableClient) CreateMicroVM(ctx context.Context, in *flintlockv1.CreateMicroVMRequest, opts ...grpc.CallOption) (*flintlockv1.CreateMicroVMResponse, error) { //nolint:lll // it would make it less readable
	return c.flintlockClient.CreateMicroVM(ctx, in, opts...)
}

func (c *disposableClient) DeleteMicroVM(ctx context.Context, in *flintlockv1.DeleteMicroVMRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) { //nolint:lll // it would make it less readable
	return c.flintlockClient.DeleteMicroVM(ctx, in, opts...)
}

func (c *disposableClient) GetMicroVM(ctx context.Context, in *flintlockv1.GetMicroVMRequest, opts ...grpc.CallOption) (*flintlockv1.GetMicroVMResponse, error) { //nolint:lll // it would make it less readable
	return c.flintlockClient.GetMicroVM(ctx, in, opts...)
}

func (c *disposableClient) ListMicroVMs(ctx context.Context, in *flintlockv1.ListMicroVMsRequest, opts ...grpc.CallOption) (*flintlockv1.ListMicroVMsResponse, error) { //nolint:lll // it would make it less readable
	return c.flintlockClient.ListMicroVMs(ctx, in, opts...)
}

func (c *disposableClient) ListMicroVMsStream(ctx context.Context, in *flintlockv1.ListMicroVMsRequest, opts ...grpc.CallOption) (flintlockv1.MicroVM_ListMicroVMsStreamClient, error) { //nolint:lll // it would make it less readable
	return c.flintlockClient.ListMicroVMsStream(ctx, in, opts...)
}
