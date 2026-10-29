// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package discover

import (
	"strings"

	"github.com/larksuite/cli/internal/apicatalog"
	"github.com/larksuite/cli/internal/cmdmeta"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/meta"
	"github.com/larksuite/cli/internal/schema"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Plans are an explicit projection: normal discovery and --flags keep their
// contracts. Root flags stay available through --flags instead of being paid
// for on every command lookup.
type commandPlan struct {
	Description string           `json:"description"`
	Usage       string           `json:"usage"`
	Risk        string           `json:"risk,omitempty"`
	Identities  []string         `json:"identities,omitempty"`
	Flags       []flag           `json:"flags,omitempty"`
	Example     string           `json:"example,omitempty"`
	Guidance    *meta.Affordance `json:"guidance,omitempty"`
	Request     *schema.Envelope `json:"request,omitempty"`
}

func describePlans(c *cobra.Command, f *cmdutil.Factory, root *cobra.Command, entries []entry) error {
	projection := c.Flag("result-fields")
	if projection == nil || !projection.Changed {
		return nil
	}
	wanted := false
	for _, field := range strings.Split(projection.Value.String(), ",") {
		field = strings.TrimSpace(field)
		if field == "plan" || strings.HasPrefix(field, "plan.") {
			wanted = true
		}
	}
	if !wanted {
		return nil
	}
	for i := range entries {
		e := &entries[i]
		cmd := e.command
		local, inherited := splitPublicFlags(cmd)
		exampleEntry := *e
		exampleEntry.Flags = describeFlags(append(append([]*pflag.Flag(nil), local...), inherited...))
		for _, fl := range inherited {
			if root.PersistentFlags().Lookup(fl.Name) != fl {
				local = append(local, fl) // Domain-specific inherited options still matter.
			}
		}
		plan := &commandPlan{Description: e.Description, Usage: cmd.UseLine(), Risk: e.Risk,
			Identities: e.Identities, Flags: describeFlags(local), Example: routingExample(root.Name(), f, exampleEntry)}
		if cmd.Long != "" {
			plan.Description = cmd.Long
		}
		if service, method, ok := cmdmeta.AffordanceRef(cmd); ok && f != nil {
			if raw, found := f.Affordance.For(service, method); found {
				if guidance, valid := (meta.Method{Affordance: raw}).ParsedAffordance(); valid {
					plan.Guidance = &guidance
				}
			}
		}
		if source, _ := cmdmeta.SourceOf(cmd); source == cmdmeta.SourceService && f != nil {
			target, err := f.APICatalog.Resolve(strings.Fields(e.Command))
			if err == nil && target.Kind == apicatalog.TargetMethod {
				request := schema.InputEnvelopeOf(f.Affordance, *target.Method)
				plan.Request = &request
				plan.Guidance = nil // The request already carries the complete overlay.
			}
			if err := f.APICatalog.Err(); err != nil {
				return err
			}
		}
		e.Plan = plan
	}
	return nil
}
