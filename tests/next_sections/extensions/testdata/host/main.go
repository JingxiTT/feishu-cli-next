// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

// This fixture is compiled as an external module. Only public integration APIs
// are available, just as for a customer embedding the CLI.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/larksuite/cli/cmd"
	"github.com/larksuite/cli/extension/command"
	"github.com/larksuite/cli/extension/credential"
	"github.com/larksuite/cli/extension/fileio"
	"github.com/larksuite/cli/extension/transport"
)

type accountProvider struct{}

func (accountProvider) Name() string { return "synthetic-host" }
func (accountProvider) ResolveAccount(context.Context) (*credential.Account, error) {
	return &credential.Account{AppID: "synthetic-host-app", Brand: credential.BrandFeishu, DefaultAs: credential.IdentityBot, SupportedIdentities: credential.SupportsBot}, nil
}
func (accountProvider) ResolveToken(_ context.Context, spec credential.TokenSpec) (*credential.Token, error) {
	if spec.Type != credential.TokenTypeTAT {
		return nil, nil
	}
	return &credential.Token{Value: "synthetic-host-token", Source: "host-fixture"}, nil
}

type redirect struct{ target *url.URL }

func (r redirect) Name() string                                             { return "synthetic-host" }
func (r redirect) ResolveInterceptor(context.Context) transport.Interceptor { return r }
func (r redirect) PreRoundTrip(req *http.Request) func(*http.Response, error) {
	req.URL.Scheme, req.URL.Host = r.target.Scheme, r.target.Host
	return nil
}

// The host owns a logical, in-memory backend. Artifact code must use this
// provider, including its optional exclusive capability, never a local fallback.
type memoryFiles struct {
	Files map[string]string `json:"files"`
	Saves int               `json:"saves"`
}

func (*memoryFiles) Open(string) (fileio.File, error)     { return nil, fs.ErrNotExist }
func (*memoryFiles) Stat(string) (fileio.FileInfo, error) { return nil, fs.ErrNotExist }
func (*memoryFiles) ResolvePath(path string) (string, error) {
	if strings.HasPrefix(path, "denied") {
		return "", &fileio.PathValidationError{Err: errors.New("host policy")}
	}
	return "memory://" + strings.TrimPrefix(path, "memory://"), nil
}
func (*memoryFiles) Save(string, fileio.SaveOptions, io.Reader) (fileio.SaveResult, error) {
	panic("artifact must use exclusive commit")
}

type saved int64

func (s saved) Size() int64 { return int64(s) }

type exclusiveFiles struct{ *memoryFiles }

func (f exclusiveFiles) SaveExclusive(path string, _ fileio.SaveOptions, body io.Reader) (fileio.SaveResult, error) {
	f.Saves++
	if os.Getenv("HOST_FILES") == "race" {
		f.Files[path] = "concurrent owner"
		return nil, fs.ErrExist
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	f.Files[path] = string(data)
	return saved(len(data)), nil
}

type filesProvider struct{ files fileio.FileIO }

func (filesProvider) Name() string                                  { return "synthetic-memory" }
func (p filesProvider) ResolveFileIO(context.Context) fileio.FileIO { return p.files }

type args struct{}
type data struct {
	ID      string      `json:"id" schema:"required" doc:"fixture value"`
	Title   string      `json:"title" schema:"required" doc:"fixture value"`
	Content string      `json:"content" schema:"required" doc:"fixture value"`
	Number  json.Number `json:"number" schema:"required" doc:"fixture value"`
}

func business(name string, risk command.Risk, request command.Request) command.Command {
	return command.Define(command.Definition[args, data]{
		Metadata: command.CommandMetadata{Service: command.DomainIm, Command: name, Description: "Synthetic host resource", Risk: risk,
			Authorization: command.AuthorizationDefinition{Identities: map[command.Identity]command.IdentityAuthorization{command.IdentityBot: {}}}},
		Hooks: command.Hooks[args, data]{
			DryRun: func(context.Context, command.CommandContext, *args) *command.DryRun {
				return command.NewDryRun(request)
			},
			Execute: func(ctx context.Context, host command.CommandContext, _ *args) (command.Result[data], error) {
				result, err := command.CallJSON[data](ctx, host, request)
				if err != nil {
					return command.Result[data]{}, err
				}
				return command.Success(result), nil
			},
		},
	})
}

func main() {
	target, err := url.Parse(os.Getenv("HOST_BACKEND"))
	if err != nil || target.Hostname() != "127.0.0.1" {
		panic("fixture requires local synthetic backend")
	}
	credential.Register(accountProvider{})
	transport.Register(redirect{target: target})
	memory := &memoryFiles{Files: make(map[string]string)}
	var files fileio.FileIO = exclusiveFiles{memory}
	if os.Getenv("HOST_FILES") == "legacy" {
		files = memory
	}
	fileio.Register(filesProvider{files: files})
	options := []cmd.BuildOption{cmd.WithoutPlugins(), cmd.WithoutStrictMode(), cmd.WithoutServiceCommands(),
		cmd.WithCommandSets(command.Set{Domain: command.ExtendDomain(command.DomainIm), Commands: []command.Command{
			business("+host-read", command.RiskRead, command.GET("/open-apis/im/v1/host-fixture")),
			business("+host-write", command.RiskWrite, command.POST("/open-apis/im/v1/host-fixture").Body(map[string]any{"title": "synthetic"})),
		}}),
	}
	if os.Getenv("HOST_AGENT") != "legacy" {
		options = append(options, cmd.WithAgentMode())
	}
	exit := cmd.ExecuteWithOptions(options...)
	// Test evidence is separate from the CLI's stdout/stderr. This local-only
	// fixture report is selected by the harness in its private temporary directory.
	report, err := json.Marshal(memory)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(os.Getenv("HOST_REPORT"), report, 0600); err != nil {
		panic(err)
	}
	os.Exit(exit)
}
