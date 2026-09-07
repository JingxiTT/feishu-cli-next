// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package contact

import (
	"context"
	"os"
	"testing"
	"time"

	clie2e "github.com/larksuite/cli/tests/cli_e2e"
	"github.com/tidwall/gjson"
)

// This opt-in workflow only reads. It never creates content or resolves a
// recipient for delivery. The coordinator supplies the independently established
// account/tenant expectation; a display name is not used as identity evidence.
func TestContactNextLiveCurrentAccountReads(t *testing.T) {
	if os.Getenv("NEXT_CONTACT_LIVE") != "1" {
		t.Skip("requires explicitly selected, account-bound live batch")
	}
	expectedID := os.Getenv("NEXT_CONTACT_LIVE_ACCOUNT_ID")
	expectedTenant := os.Getenv("NEXT_CONTACT_LIVE_TENANT_KEY")
	if expectedID == "" || expectedTenant == "" {
		t.Fatal("live fixture requires independently verified current-account ID and tenant key")
	}
	clie2e.SkipWithoutUserToken(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	read := func(args ...string) string {
		t.Helper()
		result, err := clie2e.RunCmd(ctx, clie2e.Request{Args: args, DefaultAs: "user"})
		// Do not include a personal profile or credential diagnostics in evidence.
		if err != nil || result.ExitCode != 0 || !gjson.Get(result.Stdout, "ok").Bool() {
			t.Fatal("account-bound read failed; inspect private local diagnostics before retrying")
		}
		return result.Stdout
	}
	self := read("contact", "+get-user")
	if gjson.Get(self, "data.user.open_id").String() != expectedID || gjson.Get(self, "data.user.tenant_key").String() != expectedTenant {
		t.Fatal("current-account or tenant binding mismatch; no follow-up reads performed")
	}
	lookup := read("contact", "+get-user", "--user-id", expectedID)
	if gjson.Get(lookup, "data.user.user_id").String() != expectedID {
		t.Fatal("get-user did not round-trip current account")
	}
	search := read("contact", "+search-user", "--user-ids", expectedID, "--page-size", "1")
	if !gjson.Get(search, "data.users").IsArray() || !gjson.Get(search, "data.has_more").Exists() {
		t.Fatal("search extent or array shape missing")
	}
	for _, user := range gjson.Get(search, "data.users").Array() {
		if user.Get("open_id").String() != expectedID {
			t.Fatal("ID-restricted search returned a different account")
		}
	}
	if len(gjson.Get(search, "data.users").Array()) == 0 {
		t.Log("self search returned no matches; identity-resolution completeness remains unverified")
	}
	profiles := read("contact", "user_profiles", "batch_query", "--user-id-type", "open_id", "--data", `{"user_ids":["`+expectedID+`"],"query_option":{"include_personal_status":true,"include_description":true}}`)
	if !gjson.Get(profiles, "data.user_profiles").IsArray() {
		t.Fatal("personal profiles array missing")
	}
	for _, profile := range gjson.Get(profiles, "data.user_profiles").Array() {
		if profile.Get("user_id").String() != expectedID {
			t.Fatal("profile response contains a different account")
		}
	}
	bots := read("contact", "+search-bot", "--query", "next-contact-read-fixture", "--page-size", "1")
	if !gjson.Get(bots, "data.bots").IsArray() || !gjson.Get(bots, "data.has_more").Exists() {
		t.Fatal("bot search extent or array shape missing")
	}
	t.Log("current-account read-only live batch completed; no delivery performed; bot-identity tenant binding and user-search completeness are separate checks")
}
