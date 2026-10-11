// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import (
	"net/http"

	"github.com/larksuite/cli/internal/cmdutil"
)

// Gateway calls remain fully mocked; existing presigned transfer fixtures own
// loopback servers. This preserves those tests when transfers inherit Factory.
func allowAppsLoopbackTransfers(factory *cmdutil.Factory) {
	previous := factory.HttpClient
	factory.HttpClient = func() (*http.Client, error) {
		base, err := previous()
		if err != nil {
			return nil, err
		}
		clone := *base
		clone.Transport = appsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if host := req.URL.Hostname(); host == "127.0.0.1" || host == "::1" {
				return http.DefaultTransport.RoundTrip(req)
			}
			return base.Transport.RoundTrip(req)
		})
		return &clone, nil
	}
}
