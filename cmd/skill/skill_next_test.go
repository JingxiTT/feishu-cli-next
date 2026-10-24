// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package skill

import (
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/larksuite/cli/errs"
)

const sectionFixture = "---\n" +
	"name: demo\n" +
	"# yaml comment, not a heading\n" +
	"---\n" +
	"# Title\n" +
	"\n" +
	"intro\n" +
	"## Setup\n" +
	"text\n" +
	"```bash\n" +
	"# comment in code, not a heading\n" +
	"```\n" +
	"### Setup detail\n" +
	"more\n" +
	"## Usage\n" +
	"u\n" +
	"## Usage\n" +
	"dup\n" +
	"## 概述 Overview\n" +
	"cn\n" +
	"### 字段 公式\n" +
	"cn2\n" +
	"## Use: Advanced (v2)!\n" +
	"last\n"

func sectionFS(body string) fstest.MapFS {
	return fstest.MapFS{"demo/SKILL.md": {Data: []byte(body)}}
}

func validationErr(t *testing.T, err error) *errs.ValidationError {
	t.Helper()
	var ve *errs.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *errs.ValidationError, got %T: %v", err, err)
	}
	return ve
}

func TestOutlineFixture(t *testing.T) {
	stdout, _, err := run(t, sectionFS(sectionFixture), "read", "demo", "--outline")
	if err != nil {
		t.Fatalf("outline error: %v", err)
	}
	setup := "## Setup\ntext\n```bash\n# comment in code, not a heading\n```\n### Setup detail\nmore\n"
	usage := "## Usage\nu\n"
	title := sectionFixture[strings.Index(sectionFixture, "# Title"):]
	want := strings.Join([]string{
		"Title  [title]  " + strconv.Itoa(len(title)) + " B",
		"  Setup  [setup]  " + strconv.Itoa(len(setup)) + " B",
		"    Setup detail  [setup-detail]  " + strconv.Itoa(len("### Setup detail\nmore\n")) + " B",
		"  Usage  [usage]  " + strconv.Itoa(len(usage)) + " B",
		"  Usage  [usage-2]  " + strconv.Itoa(len("## Usage\ndup\n")) + " B",
		"  概述 Overview  [概述-overview]  " + strconv.Itoa(len("## 概述 Overview\ncn\n### 字段 公式\ncn2\n")) + " B",
		"    字段 公式  [字段-公式]  " + strconv.Itoa(len("### 字段 公式\ncn2\n")) + " B",
		"  Use: Advanced (v2)!  [use-advanced-v2]  " + strconv.Itoa(len("## Use: Advanced (v2)!\nlast\n")) + " B",
		"",
	}, "\n")
	if stdout != want {
		t.Errorf("outline mismatch\n got:\n%s\nwant:\n%s", stdout, want)
	}
}

func TestSectionBySlugTextAndSubstring(t *testing.T) {
	cases := []struct {
		query string
		want  string
	}{
		{"setup-detail", "### Setup detail\nmore\n"},
		{"Setup", "## Setup\ntext\n```bash\n# comment in code, not a heading\n```\n### Setup detail\nmore\n"},
		{"detail", "### Setup detail\nmore\n"},
		{"usage-2", "## Usage\ndup\n"},
		{"字段-公式", "### 字段 公式\ncn2\n"},
		{"公式", "### 字段 公式\ncn2\n"},
		{"advanced", "## Use: Advanced (v2)!\nlast\n"},
	}
	for _, c := range cases {
		stdout, _, err := run(t, sectionFS(sectionFixture), "read", "demo", "--section", c.query)
		if err != nil {
			t.Fatalf("--section %q error: %v", c.query, err)
		}
		if stdout != c.want {
			t.Errorf("--section %q: got %q want %q", c.query, stdout, c.want)
		}
	}
}

func TestSectionRepeatableAndCommaSeparatedInDocumentOrder(t *testing.T) {
	stdout, _, err := run(t, sectionFS(sectionFixture), "read", "demo", "--section", "usage-2,setup-detail", "--section", "title")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !strings.HasPrefix(stdout, "# Title\n") {
		t.Errorf("expected document order starting at Title, got %q", stdout)
	}
	if strings.Count(stdout, "Setup detail") != 1 {
		t.Errorf("child section emitted twice inside parent: %q", stdout)
	}
}

func TestSectionAmbiguousAndNotFound(t *testing.T) {
	_, _, err := run(t, sectionFS(sectionFixture), "read", "demo", "--section", "set")
	ve := validationErr(t, err)
	if !strings.Contains(ve.Hint, "setup-detail") || !strings.Contains(ve.Hint, "setup (Setup)") {
		t.Errorf("ambiguous hint should list candidates with slugs, got %q", ve.Hint)
	}

	_, _, err = run(t, sectionFS(sectionFixture), "read", "demo", "--section", "nonexistent")
	ve = validationErr(t, err)
	if !strings.Contains(ve.Hint, "usage-2") {
		t.Errorf("not-found hint should list candidates, got %q", ve.Hint)
	}
}

func TestSectionHintCapsCandidates(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 30; i++ {
		b.WriteString("## Topic\nx\n")
	}
	_, _, err := run(t, sectionFS(b.String()), "read", "demo", "--section", "opi")
	ve := validationErr(t, err)
	if !strings.Contains(ve.Hint, "and 10 more") {
		t.Errorf("expected cap at 20 with overflow count, got %q", ve.Hint)
	}
}

func TestOutlineAndSectionConflict(t *testing.T) {
	_, _, err := run(t, sectionFS(sectionFixture), "read", "demo", "--outline", "--section", "title")
	validationErr(t, err)
}

func TestReadWithoutFlagsIsByteIdentical(t *testing.T) {
	stdout, stderr, err := run(t, sectionFS(sectionFixture), "read", "demo")
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if stdout != sectionFixture {
		t.Errorf("stdout not byte-identical to file")
	}
	if strings.Contains(stderr, "Tip: this file is") {
		t.Errorf("small file must not emit size tip: %q", stderr)
	}
}

func TestLargeReadTipGoesToStderrOnly(t *testing.T) {
	big := sectionFixture + strings.Repeat("filler line\n", 1000)
	stdout, stderr, err := run(t, sectionFS(big), "read", "demo")
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if stdout != big {
		t.Errorf("stdout must stay byte-identical for large files")
	}
	if !strings.Contains(stderr, "--outline") || !strings.Contains(stderr, "--section") {
		t.Errorf("expected stderr tip naming --outline and --section, got %q", stderr)
	}

	stdout, stderr, err = run(t, sectionFS(big), "read", "demo", "--section", "title")
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if strings.Contains(stderr, "Tip: this file is") {
		t.Errorf("tip must not repeat when a section is requested: %q", stderr)
	}

	refFS := fstest.MapFS{"demo/references/big.md": {Data: []byte(big)}}
	_, stderr, err = run(t, refFS, "read", "demo", "references/big.md")
	if err != nil {
		t.Fatalf("reference read error: %v", err)
	}
	if stderr != "" {
		t.Errorf("reference reads must keep stderr empty, got %q", stderr)
	}
}

func TestSectionWithJSONEnvelope(t *testing.T) {
	stdout, _, err := run(t, sectionFS(sectionFixture), "read", "demo", "--section", "usage-2", "--json")
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	var env struct {
		Content string `json:"content"`
	}
	if e := json.Unmarshal([]byte(stdout), &env); e != nil {
		t.Fatalf("invalid JSON: %v\n%s", e, stdout)
	}
	if env.Content != "## Usage\ndup\n" {
		t.Errorf("JSON content should be the selected section, got %q", env.Content)
	}
}

func TestRealEmbeddedSkillOutlineAndSection(t *testing.T) {
	fsys := os.DirFS("../../skills")
	if _, err := os.Stat("../../skills/lark-base/references/lark-base-field-formula.md"); err != nil {
		t.Skipf("skill source tree unavailable: %v", err)
	}
	const file = "references/lark-base-field-formula.md"
	full, _, err := run(t, fsys, "read", "lark-base", file)
	if err != nil {
		t.Fatalf("full read error: %v", err)
	}
	outline, _, err := run(t, fsys, "read", "lark-base", file, "--outline")
	if err != nil {
		t.Fatalf("outline error: %v", err)
	}
	const line = "Section 8: Complete Function Reference  [section-8-complete-function-reference]  "
	i := strings.Index(outline, line)
	if i < 0 {
		t.Fatalf("outline lacks Section 8 heading:\n%s", outline)
	}
	size := strings.TrimSuffix(strings.SplitN(outline[i+len(line):], "\n", 2)[0], " B")
	section, _, err := run(t, fsys, "read", "lark-base", file, "--section", "section-8-complete-function-reference")
	if err != nil {
		t.Fatalf("section error: %v", err)
	}
	if len(section) >= len(full) || len(section) == 0 {
		t.Fatalf("section should be a strict subset: section=%d full=%d", len(section), len(full))
	}
	if size != strconv.Itoa(len(section)) {
		t.Errorf("outline size %s does not match section bytes %d", size, len(section))
	}
}
