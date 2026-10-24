// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package markdown

import (
	"context"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/httpmock"
	"github.com/larksuite/cli/shortcuts/common"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestMarkdownPatchIdenticalReplacementDoesNotUpload(t *testing.T) {
	for _, tc := range []struct {
		name, pattern, replacement string
		regex                      bool
	}{
		{"literal", "中文 😀", "中文 😀", false},
		{"capture", "(中文 😀)", "$1", true},
		{"zero_width", "^", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, stdout, _, reg := cmdutil.TestFactory(t, markdownTestConfig())
			original := "# 中文 😀\r\n\\*literal\\* & <tag>\n"
			reg.Register(&httpmock.Stub{Method: "GET", URL: "/open-apis/drive/v1/medias/box_integrity/preview_download?preview_type=16", Status: 200, RawBody: []byte(original)})
			args := []string{"+patch", "--file-token", "box_integrity", "--pattern", tc.pattern, "--content", tc.replacement}
			if tc.regex {
				args = append(args, "--regex")
			}
			require.NoError(t, mountAndRunMarkdown(t, MarkdownPatch, args, f, stdout))
			data := decodeMarkdownEnvelope(t, stdout)
			require.False(t, common.GetBool(data, "updated"))
			require.Equal(t, 1, common.GetInt(data, "match_count"))
			require.Equal(t, len(original), common.GetInt(data, "size_bytes_after"))
		})
	}
}

func TestMarkdownUnknownUploadOutcomeDoesNotRetry(t *testing.T) {
	for _, failure := range []error{
		errs.NewNetworkError(errs.SubtypeNetworkTransport, "response lost"),
		errs.NewNetworkError(errs.SubtypeNetworkServer, "gateway failed").WithCode(502).WithRetryable(),
		errs.NewAPIError(errs.SubtypeUnknown, "backend failure").WithCode(10002).WithRetryable(),
	} {
		f, _, _, _ := cmdutil.TestFactory(t, markdownTestConfig())
		rt := common.TestNewRuntimeContextForAPI(context.Background(), &cobra.Command{Use: "+create"}, markdownTestConfig(), f, core.AsUser)
		t.Run(failure.Error(), func(t *testing.T) {
			for _, stage := range []string{"all", "prepare", "part"} {
				attempts := 0
				var err error
				switch stage {
				case "all":
					_, err = withMarkdownUploadRetryResult(rt, markdownUploadAllAction, func() (markdownUploadResult, error) { attempts++; return markdownUploadResult{}, failure })
				case "prepare":
					_, err = withMarkdownUploadRetryData(rt, markdownUploadPrepareAction, func() (map[string]interface{}, error) { attempts++; return nil, failure })
				case "part":
					err = withMarkdownUploadRetryVoid(rt, "upload part", func() error { attempts++; return failure })
				}
				require.Same(t, failure, err)
				require.Equal(t, 1, attempts, stage)
			}
		})
	}
}
