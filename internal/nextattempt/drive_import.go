// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package nextattempt

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type driveImportPoint struct {
	Type *int    `json:"mount_type"`
	Key  *string `json:"mount_key"`
}

type driveImportSource struct {
	FileToken string            `json:"file_token"`
	Extension string            `json:"file_extension"`
	FileName  string            `json:"file_name"`
	Type      string            `json:"type"`
	Target    string            `json:"token"`
	Point     *driveImportPoint `json:"point"`
}

type driveImportAttempt struct {
	Binding Binding
	Origin  string
	Ticket  string
	Source  driveImportSource
}

// DriveImportCompletionMatches validates native completion against the exact
// create body. It has no journal access and cannot upgrade mutation evidence.
func DriveImportCompletionMatches(ticket string, requestBody, responseData []byte) bool {
	if ticket == "" || len(requestBody) > responseEvidenceLimit || len(responseData) > responseEvidenceLimit {
		return false
	}
	var source driveImportSource
	if json.Unmarshal(requestBody, &source) != nil || !source.valid() {
		return false
	}
	return driveImportCompleted(driveImportAttempt{Ticket: ticket, Source: source}, responseData)
}

func (source driveImportSource) valid() bool {
	return source.FileToken != "" && source.Extension != "" && source.Type != "" && source.Point != nil && source.Point.Type != nil && source.Point.Key != nil
}

func (j *Journal) captureDriveImport(receipt Receipt, req *http.Request, data json.RawMessage) bool {
	var source driveImportSource
	var result struct {
		Ticket string `json:"ticket"`
	}
	if !decodeRequestEvidence(req, &source) || json.Unmarshal(data, &result) != nil || result.Ticket == "" || !source.valid() {
		return false
	}
	origin := req.URL.Scheme + "://" + req.URL.Host
	size := len(origin) + len(result.Ticket) + len(source.FileToken) + len(source.Extension) + len(source.FileName) + len(source.Type) + len(source.Target) + len(*source.Point.Key)
	j.mu.Lock()
	defer j.mu.Unlock()
	// Each +import creates one task. Keep a bounded allowance for composed
	// workflows without retaining unbounded source/destination identifiers.
	if len(j.driveImports) >= 128 || size > (4<<20)-j.driveImportSize {
		return false
	}
	if j.driveImports == nil {
		j.driveImports = make(map[string]driveImportAttempt)
	}
	j.driveImports[receipt.ID] = driveImportAttempt{Binding: receipt.Binding, Origin: origin, Ticket: result.Ticket, Source: source}
	j.driveImportSize += size
	return true
}

// HasDriveImport reports only same-invocation captured create acknowledgements;
// standalone status reads never resolve identity or touch receipt storage.
func (j *Journal) HasDriveImport(ticket string) bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	for _, attempt := range j.driveImports {
		if attempt.Ticket == ticket {
			return true
		}
	}
	return false
}

// ReconcileDriveImport consumes native completion proof from a successful GET
// of this exact ticket, made using the same runtime identity and origin. Only
// a captured pending import can change; a ticket by itself is never evidence.
// Failed persistence leaves the receipt unknown and marks the invocation failed
// without causing the caller to poll or replay an already completed operation.
func (j *Journal) ReconcileDriveImport(store Store, binding Binding, origin, ticket string, data []byte) {
	if len(data) > responseEvidenceLimit {
		return
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	id := ""
	var attempt driveImportAttempt
	for candidate, pending := range j.driveImports {
		if pending.Ticket == ticket && pending.Binding == binding && pending.Origin == origin {
			if id != "" {
				return // Reused tickets cannot identify one create attempt.
			}
			id, attempt = candidate, pending
		}
	}
	if id == "" || !driveImportCompleted(attempt, data) {
		return
	}
	for i, receipt := range j.receipts {
		if receipt.ID != id || receipt.Outcome != Unknown || receipt.Evidence != "drive_import_task_pending" || receipt.PersistenceFailed {
			continue
		}
		completed := receipt
		completed.Outcome, completed.Evidence = Applied, "drive_import_task_completed"
		completed.UpdatedAt = time.Now().UTC()
		if persistDriveImportCompletion(store, receipt, completed) == nil {
			j.receipts[i] = completed
			return
		}
		j.receipts[i].PersistenceFailed = true
		return
	}
}

func persistDriveImportCompletion(store Store, pending, completed Receipt) error {
	if err := store.validateDir(false); err != nil {
		return err
	}
	lock, err := store.lock(pending.ID)
	if err != nil {
		return err
	}
	defer lock.Unlock()
	previous, err := store.read(pending.ID)
	if err != nil {
		return err
	}
	if previous.Binding != pending.Binding || previous.Outcome != Unknown || previous.Evidence != pending.Evidence || previous.Method != "POST" {
		return invalidState()
	}
	return store.finish(completed)
}

func driveImportCompleted(attempt driveImportAttempt, data []byte) bool {
	var envelope struct {
		Result json.RawMessage `json:"result"`
	}
	var result struct {
		Status    *int            `json:"job_status"`
		Type      string          `json:"type"`
		Token     string          `json:"token"`
		Ticket    json.RawMessage `json:"ticket"`
		FileToken json.RawMessage `json:"file_token"`
		Extension json.RawMessage `json:"file_extension"`
		FileName  json.RawMessage `json:"file_name"`
		Point     json.RawMessage `json:"point"`
		Error     json.RawMessage `json:"job_error_msg"`
	}
	if json.Unmarshal(data, &envelope) != nil || json.Unmarshal(envelope.Result, &result) != nil || result.Status == nil || *result.Status != 0 || result.Type != attempt.Source.Type || strings.TrimSpace(result.Token) == "" || attempt.Source.Target != "" && result.Token != attempt.Source.Target {
		return false
	}
	for _, echo := range []struct {
		raw  json.RawMessage
		want string
	}{{result.Ticket, attempt.Ticket}, {result.FileToken, attempt.Source.FileToken}, {result.Extension, attempt.Source.Extension}, {result.FileName, attempt.Source.FileName}, {result.Error, ""}} {
		if len(echo.raw) == 0 {
			continue
		}
		var value *string
		if json.Unmarshal(echo.raw, &value) != nil || value == nil || *value != echo.want {
			return false
		}
	}
	if len(result.Point) > 0 {
		var point map[string]json.RawMessage
		if json.Unmarshal(result.Point, &point) != nil || point == nil {
			return false
		}
		for key, raw := range point {
			switch key {
			case "mount_type":
				var value *int
				if json.Unmarshal(raw, &value) != nil || value == nil || *value != *attempt.Source.Point.Type {
					return false
				}
			case "mount_key":
				var value *string
				if json.Unmarshal(raw, &value) != nil || value == nil || *value != *attempt.Source.Point.Key {
					return false
				}
			default:
				return false
			}
		}
	}
	return true
}
