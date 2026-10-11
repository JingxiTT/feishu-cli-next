// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmd

import (
	"io/fs"
	"strings"

	"github.com/larksuite/cli/cmd/discover"
	"github.com/larksuite/cli/cmd/guide"
	"github.com/larksuite/cli/cmd/receipts"
	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/cmdmeta"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/output"
	"github.com/spf13/cobra"
)

// WithAgentMode enables the next distribution's shared interface. Existing SDK
// embedders retain their command and wire contracts unless they opt in.
func WithAgentMode() BuildOption { return func(c *buildConfig) { c.agentMode = true } }

var agentSkillContent fs.FS

// SetAgentSkillContent supplies the single project-installable entry skill.
// Upstream embedded skills remain available as on-demand compatibility content.
func SetAgentSkillContent(content fs.FS) { agentSkillContent = content }

func configureAgentFoundation(f *cmdutil.Factory, root *cobra.Command) *output.AgentOptions {
	opts := &output.AgentOptions{}
	f.EnableAgentAttempts = true
	f.IOStreams.Out = output.WithAgentOptions(f.IOStreams.Out, opts)
	root.Use = "feishu-cli-next"
	root.Short = "Feishu/Lark commands for agents and shell workflows"
	root.SetVersionTemplate("{{.Name}} {{.Version}} (upstream: larksuite/cli@v1.0.97)\n")
	root.AddCommand(discover.NewCmd(f, root), guide.NewCmd(f, agentSkillContent), receipts.NewCmdReceipts(f))
	root.PersistentFlags().StringVar(&opts.Fields, "result-fields", "", "select JSON result fields; keep outcome and pagination metadata")
	root.PersistentFlags().BoolVar(&opts.Detail, "result-detail", false, "indent full JSON results")
	root.PersistentFlags().StringVar(&opts.SaveResult, "save-result", "", "save a complete read result as a versioned artifact (exclusive create)")
	opts.ResolveBinding = func() (output.ArtifactBinding, error) {
		ctx := root.Context()
		if f.CurrentCommand != nil {
			ctx = f.CurrentCommand.Context()
		}
		return f.ResolveArtifactBinding(ctx, f.ResolvedIdentity)
	}
	opts.ResolveMutation = func() (string, []string, bool) {
		summary := f.AgentAttemptSummary()
		ids := make([]string, 0, len(summary.Receipts))
		for _, receipt := range summary.Receipts {
			ids = append(ids, receipt.ID)
		}
		return summary.Outcome, ids, summary.PersistenceFailed
	}
	return opts
}

func finalizeAgentFoundation(f *cmdutil.Factory, root *cobra.Command, opts *output.AgentOptions) {
	root.Long = "Use domain commands, generated API methods, or raw API calls. `discover [query]` searches this build's commands; `guide` shows the English entry skill. JSON results are compact, with full data available by default."
	root.SetUsageTemplate(rootUsageTemplatePrefix + "Usage:\n  feishu-cli-next <command> [flags]" + rootUsageTemplateSuffix + "\n")
	// A development distribution must never upgrade itself from the upstream
	// npm package. Packaging supplies a next release channel in its own section.
	if update, _, err := root.Find([]string{"update"}); err == nil && update.Name() == "update" {
		update.Short = "Report this source build's update-channel status"
		update.Long = "This source build has no feishu-cli-next release channel. Rebuild this checkout with make build. Project skill updates use guide update."
		update.RunE = func(c *cobra.Command, args []string) error {
			check, _ := c.Flags().GetBool("check")
			if check {
				return output.WriteJSON(f.IOStreams.Out, map[string]any{"ok": true, "data": map[string]any{"distribution": "feishu-cli-next", "update_channel": "unconfigured"}})
			}
			return errs.NewValidationError(errs.SubtypeFailedPrecondition, "this source build has no feishu-cli-next release channel").WithHint("Rebuild this checkout with make build; use guide update for the project skill.")
		}
	}
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		// Cobra otherwise replays pflag deprecation notices to command stdout.
		// Keep parser diagnostics separate from structured business results.
		c.Flags().SetOutput(f.IOStreams.ErrOut)
		c.PersistentFlags().SetOutput(f.IOStreams.ErrOut)
		// These are repository-owned help strings, never user data or results.
		c.Short = strings.ReplaceAll(c.Short, "lark-cli", "feishu-cli-next")
		c.Long = strings.ReplaceAll(c.Long, "lark-cli", "feishu-cli-next")
		c.Example = strings.ReplaceAll(c.Example, "lark-cli", "feishu-cli-next")
		if c.RunE != nil {
			run := c.RunE
			c.RunE = func(command *cobra.Command, args []string) error {
				if err := validateAgentOutput(f, command, opts); err != nil {
					return err
				}
				runErr := run(command, args)
				if runErr == nil {
					runErr = opts.Finish()
				}
				return f.FinalizeAgentAttempt(runErr)
			}
		}
		for _, child := range c.Commands() {
			walk(child)
		}
	}
	walk(root)
}

func validateAgentOutput(f *cmdutil.Factory, c *cobra.Command, opts *output.AgentOptions) error {
	if opts.Fields != "" || opts.SaveResult != "" {
		dryRun, _ := c.Flags().GetBool("dry-run")
		for _, name := range []string{"raw", "jq", "stream", "concise"} {
			if name == "concise" && dryRun {
				continue // IM dry-run previews remain JSON even with --concise.
			}
			flag := c.Flags().Lookup(name)
			if flag != nil && flag.Changed && flag.Value.String() != "false" && flag.Value.String() != "" {
				return errs.NewValidationError(errs.SubtypeInvalidArgument, "result projection/artifacts cannot be combined with --%s", name).WithParam("--" + name)
			}
		}
		if flag := c.Flags().Lookup("format"); flag != nil && flag.Value.String() != "" && flag.Value.String() != "json" {
			return errs.NewValidationError(errs.SubtypeInvalidArgument, "result projection/artifacts require --format json").WithParam("--format")
		}
		source, _ := cmdmeta.SourceOf(c)
		business := source == cmdmeta.SourceShortcut || source == cmdmeta.SourceService || c.Name() == "api"
		if !business && c.Name() != "discover" {
			param := "--result-fields"
			if opts.SaveResult != "" {
				param = "--save-result"
			}
			return errs.NewValidationError(errs.SubtypeInvalidArgument, "result projection/artifacts are available on structured business commands").WithParam(param)
		}
		if opts.SaveResult != "" {
			risk, _ := cmdmeta.Risk(c)
			// Raw GET can issue credentials (for example API keys and Git PATs).
			// Without an operation schema, resource tokens cannot safely be
			// distinguished from secrets destined for persistent artifacts.
			if !business || c.Name() == "api" || risk != "read" {
				return errs.NewValidationError(errs.SubtypeInvalidArgument, "--save-result requires a declared, non-raw read operation").WithParam("--save-result")
			}
			opts.FileIO = f.ResolveFileIO(c.Context())
		}
	}
	return opts.Validate()
}
