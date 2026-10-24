// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package drive

import (
	"testing"

	"github.com/larksuite/cli/errs"
)

func TestImportStatusRequiresExplicitNativeStatus(t *testing.T) {
	for _, value := range []interface{}{nil, "0", false, 0.5, []interface{}{0}} {
		t.Run("malformed", func(t *testing.T) {
			_, err := parseDriveImportStatus("ticket", map[string]interface{}{"result": map[string]interface{}{"token": "doc", "type": "docx", "job_status": value}})
			problem, ok := errs.ProblemOf(err)
			if !ok || problem.Subtype != errs.SubtypeInvalidResponse {
				t.Fatalf("want invalid_response for %#v, got %v", value, err)
			}
		})
	}
	for _, result := range []map[string]interface{}{
		{"token": "doc", "type": "docx"},
		{"token": "doc", "type": "docx", "job_status": 0, "ticket": "other"},
		{"token": "doc", "type": "docx", "job_status": 0, "ticket": nil},
		{"token": "doc", "type": "docx", "job_status": 0, "ticket": ""},
		{"token": "doc", "type": "docx", "job_status": 0, "job_error_msg": nil},
		{"token": 3, "type": "docx", "job_status": 0},
		{"token": "doc", "type": []interface{}{"docx"}, "job_status": 0},
	} {
		_, err := parseDriveImportStatus("ticket", map[string]interface{}{"result": result})
		problem, ok := errs.ProblemOf(err)
		if !ok || problem.Subtype != errs.SubtypeInvalidResponse {
			t.Fatalf("want invalid_response for %#v, got %v", result, err)
		}
	}
	for _, wrapped := range []interface{}{nil, "invalid", []interface{}{}} {
		_, err := parseDriveImportStatus("ticket", map[string]interface{}{"result": wrapped, "job_status": 0, "token": "doc"})
		problem, ok := errs.ProblemOf(err)
		if !ok || problem.Subtype != errs.SubtypeInvalidResponse {
			t.Fatalf("malformed wrapper accepted: %v", err)
		}
	}

	for _, code := range []interface{}{0, float64(0)} {
		status, err := parseDriveImportStatus("ticket", map[string]interface{}{"token": "doc", "type": "docx", "job_status": code})
		if err != nil || !status.Ready() {
			t.Fatalf("explicit terminal status rejected: %v %#v", err, status)
		}
	}
	for _, token := range []string{"", " \t"} {
		status, err := parseDriveImportStatus("ticket", map[string]interface{}{"job_status": 0, "token": token})
		if err != nil || status.Ready() || !status.Pending() || status.StatusLabel() != "pending" {
			t.Fatalf("empty destination must remain pending: %v %#v", err, status)
		}
	}
	status, err := parseDriveImportStatus("ticket", map[string]interface{}{"result": map[string]interface{}{"job_status": 2}})
	if err != nil || !status.Pending() || status.Ready() {
		t.Fatalf("pending status changed: %v %#v", err, status)
	}
}
