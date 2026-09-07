// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmdutil

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/nextattempt"
)

// AgentReceiptStore locates CLI-owned evidence in the effective workspace.
func (f *Factory) AgentReceiptStore() nextattempt.Store {
	return nextattempt.Store{Dir: filepath.Join(core.GetConfigDir(), "next", "receipts")}
}
func (f *Factory) AgentAttemptSummary() nextattempt.Summary { return f.agentAttempts.Summary() }
func (f *Factory) AgentAttemptExitCode() int                { return f.AgentAttemptSummary().ExitCode() }

func (f *Factory) agentAttemptTransport(base http.RoundTripper) http.RoundTripper {
	return &nextattempt.Transport{
		Base: base, Store: f.AgentReceiptStore, Journal: &f.agentAttempts,
		IsMutation: func(req *http.Request) bool {
			if !f.EnableAgentAttempts {
				return false
			}
			switch req.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				return false
			}
			// Auth endpoints issue credentials, not user-content mutations. Excluding
			// these before ResolveAccount prevents recursion while tokens are resolved.
			path := req.URL.Path
			if strings.HasPrefix(path, "/open-apis/auth/") || strings.HasPrefix(path, "/open-apis/authen/") || strings.HasPrefix(path, "/open-apis/oauth/") {
				return false
			}
			if nextattempt.IsStoragePutRequest(req) {
				return true
			}
			if mutation, declared := declaredRequestEffect(req); declared {
				return mutation
			}
			if f.CurrentCommand != nil {
				if risk, ok := GetRisk(f.CurrentCommand); ok && risk == "read" {
					return false
				}
			}
			return true
		},
		Resolve: func(req *http.Request) (nextattempt.Binding, error) {
			cfg, err := f.Config()
			if err != nil {
				return nextattempt.Binding{}, err
			}
			identity := f.ResolvedIdentity
			if requestIdentity, ok := core.RequestIdentityFromContext(req.Context()); ok {
				identity = requestIdentity
			}
			if identity != core.AsUser && identity != core.AsBot {
				return nextattempt.Binding{}, errs.NewValidationError(errs.SubtypeFailedPrecondition, "mutation identity has not been resolved")
			}
			binding := nextattempt.Binding{Environment: core.ResolveEndpoints(cfg.Brand).Open, Account: cfg.ProfileName, AppID: cfg.AppID, Identity: string(identity)}
			if identity == core.AsUser {
				binding.UserID = cfg.UserOpenId
				if f.Credential != nil {
					resolved, err := f.Credential.ResolveUserBinding(req.Context())
					if err != nil {
						return nextattempt.Binding{}, err
					}
					binding.UserID, binding.TenantKey, binding.UserVerified = resolved.OpenID, resolved.TenantKey, resolved.Verified
					binding.TenantVerified = resolved.TenantKey != ""
				}
				if binding.UserID == "" {
					return nextattempt.Binding{}, errs.NewValidationError(errs.SubtypeFailedPrecondition, "mutation requires a resolved user identity")
				}
			}

			return binding, nil
		},
	}
}

// FinalizeAgentAttempt makes non-success mutation evidence observable to hosts
// executing a built Cobra tree. Call after the command and output finalization.
func (f *Factory) FinalizeAgentAttempt(runErr error) error {
	if !f.EnableAgentAttempts {
		return runErr
	}
	summary := f.AgentAttemptSummary()
	if len(summary.Receipts) == 0 || summary.ExitCode() == 0 {
		return runErr
	}
	ids := make([]string, len(summary.Receipts))
	for i, receipt := range summary.Receipts {
		ids[i] = receipt.ID
	}
	return errs.NewMutationOutcomeError(summary.Outcome, ids, summary.PersistenceFailed, runErr)
}
