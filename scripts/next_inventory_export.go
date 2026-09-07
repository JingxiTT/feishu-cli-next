// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

//go:build ignore

// Run by next_inventory.py in an isolated source copy. No command executes.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	rootcmd "github.com/larksuite/cli/cmd"
	"github.com/larksuite/cli/events"
	_ "github.com/larksuite/cli/extension/credential/env" // match the production entry point
	"github.com/larksuite/cli/internal/cmdmeta"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/commandbridge"
	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/event/catalog"
	"github.com/larksuite/cli/internal/flagalias"
	"github.com/larksuite/cli/internal/registry"
	"github.com/larksuite/cli/shortcuts"
	"github.com/larksuite/cli/shortcuts/common"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func main() {
	root := rootcmd.Build(context.Background(), cmdutil.InvocationContext{},
		rootcmd.WithIO(strings.NewReader(""), io.Discard, io.Discard),
		rootcmd.WithStartupBrand(core.LarkBrand(os.Getenv("LARKSUITE_CLI_BRAND"))),
		rootcmd.WithoutPlugins(), rootcmd.WithoutStrictMode(),
		rootcmd.WithServiceCatalog(registry.EmbeddedCatalog()))
	commands := map[string]any{}
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		flags := map[string]any{}
		visit := func(f *pflag.Flag) {
			flags[f.Name] = map[string]any{
				"type": f.Value.Type(), "default": f.DefValue, "no_opt_default": f.NoOptDefVal,
				"aliases": flagalias.Aliases(f), "shorthand": f.Shorthand, "hidden": f.Hidden,
				"annotations": f.Annotations,
			}
		}
		c.Flags().VisitAll(visit)
		c.InheritedFlags().VisitAll(visit)
		source, _ := cmdmeta.SourceOf(c)
		risk, _ := cmdmeta.Risk(c)
		path := strings.TrimSpace(strings.TrimPrefix(c.CommandPath(), root.Name()))
		commands[path] = map[string]any{"aliases": c.Aliases, "flags": flags,
			"runnable": c.Runnable(), "hidden": c.Hidden, "disable_flag_parsing": c.DisableFlagParsing, "source": source,
			"domain": cmdmeta.Domain(c), "risk": risk, "identities": cmdmeta.Identities(c)}
		for _, child := range c.Commands() {
			walk(child)
		}
	}
	walk(root)
	shortcutDefs := map[string]any{}
	for _, s := range shortcuts.AllShortcuts() {
		typedSchema, _ := common.ShortcutSchema(s, commandbridge.Access{})
		schemas := map[string]json.RawMessage{}
		if s.PrintFlagSchema != nil {
			for _, f := range s.Flags {
				raw, err := s.PrintFlagSchema(f.Name)
				if err == nil && json.Valid(raw) {
					schemas[f.Name] = raw
				}
			}
		}
		shortcutDefs[s.Service+" "+s.Command] = map[string]any{
			"flags": s.Flags, "typed_schema": typedSchema, "flag_schemas": schemas,
			"risk": s.Risk, "auth_types": s.AuthTypes,
			"user_scopes": s.ScopesForIdentity("user"), "bot_scopes": s.ScopesForIdentity("bot"),
			"conditional_user_scopes": s.ConditionalScopesForIdentity("user"),
			"conditional_bot_scopes":  s.ConditionalScopesForIdentity("bot"),
			"dry_run":                 s.DryRun != nil, "has_validation": s.Validate != nil,
		}
	}
	compiled, err := catalog.Compile(events.All(), catalog.StrategyRefs{catalog.StrategyNone, catalog.StrategyLegacyPreConsume})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	eventDefs := map[string]any{}
	for _, e := range compiled.Entries() {
		eventDefs[e.Descriptor().Key] = map[string]any{"descriptor": e.Descriptor(), "output": e.Output(), "capability": e.Capability()}
	}
	services := []string{}
	for _, s := range registry.EmbeddedServicesTyped() {
		services = append(services, s.Name)
	}
	sort.Strings(services)
	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{"commands": commands, "shortcuts": shortcutDefs, "events": eventDefs, "services": services}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
