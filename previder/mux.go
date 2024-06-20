package previder

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"
)

func GetMuxedProvider(ctx context.Context) (func() tfprotov6.ProviderServer, error) {

	providers := []func() tfprotov6.ProviderServer{
		providerserver.NewProtocol6(NewPreviderProvider()),
	}

	muxServer, err := tf6muxserver.NewMuxServer(ctx, providers...)
	if err != nil {
		return nil, err
	}

	return muxServer.ProviderServer, nil
}
