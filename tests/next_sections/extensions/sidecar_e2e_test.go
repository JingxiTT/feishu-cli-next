// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package extensions

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/larksuite/cli/sidecar"
	"github.com/larksuite/cli/tests/nextfixture"
)

// Both brand and identity variants use the production authsidecar build. The
// handler verifies the wire signature and never forwards the named target.
func TestSidecarArtifactBindings(t *testing.T) {
	for brand, host := range map[string]string{"feishu": "open.feishu.cn", "lark": "open.larksuite.com"} {
		for _, identity := range []string{"bot", "user"} {
			t.Run(brand+"/"+identity, func(t *testing.T) {
				var requests atomic.Int32
				f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					body, _ := io.ReadAll(r.Body)
					target, err := url.Parse(r.Header.Get(sidecar.HeaderProxyTarget))
					if err != nil {
						t.Errorf("invalid sidecar target: %v", err)
						http.Error(w, "invalid target", http.StatusBadRequest)
						return
					}
					canonical := sidecar.CanonicalRequest{
						Version: r.Header.Get(sidecar.HeaderProxyVersion), Method: r.Method,
						Host: target.Host, PathAndQuery: r.URL.RequestURI(),
						BodySHA256: sidecar.BodySHA256(body), Timestamp: r.Header.Get(sidecar.HeaderProxyTimestamp),
						Identity: r.Header.Get(sidecar.HeaderProxyIdentity), AuthHeader: r.Header.Get(sidecar.HeaderProxyAuthHeader),
					}
					if err := sidecar.Verify([]byte("synthetic-foundation-proxy-key"), canonical, r.Header.Get(sidecar.HeaderProxySignature)); err != nil {
						t.Errorf("sidecar signature: %v", err)
					}
					if canonical.Host != host || canonical.Identity != identity || canonical.Version != sidecar.ProtocolV1 || canonical.AuthHeader != "Authorization" || r.Header.Get("Authorization") != "" || r.Method != http.MethodGet || r.URL.Path != "/open-apis/drive/v1/files" {
						t.Errorf("unexpected sidecar request: %+v", canonical)
					}
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(w, `{"code":0,"data":{"files":[{"token":"file_001","name":"host fixture"}],"has_more":false}}`)
				}))
				f.Env["LARKSUITE_CLI_BRAND"] = brand
				path := filepath.Join(f.WorkDir, "artifact.json")
				result := f.Run("drive", "files", "list", "--as", identity, "--save-result", path)
				if result.ExitCode != 0 || result.Stderr != "" || requests.Load() != 1 {
					t.Fatalf("sidecar artifact: %+v requests=%d", result, requests.Load())
				}
				content, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				binding := envelope(t, string(content))["binding"].(map[string]any)
				if binding["environment"] != brand || binding["identity"] != identity || binding["app_id"] != "synthetic-foundation-app" {
					t.Fatalf("incorrect binding: %v", binding)
				}
				if identity == "user" && (binding["user_id"] != "ou_foundation_fixture" || binding["user_verified"] != true || binding["tenant_verified"] != true) {
					t.Fatalf("lost observed user evidence: %v", binding)
				}
				if identity == "bot" && (binding["user_id"] != nil || binding["tenant_verified"] != true || binding["tenant_key"] != "foundation-fixture-tenant") {
					t.Fatalf("incorrect observed bot evidence: %v", binding)
				}
			})
		}
	}
}
