// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package minutes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/validate"
	"github.com/larksuite/cli/shortcuts/common"
)

const (
	minutesWordReplaceNoEditPermission  = 2091005
	minutesWordReplaceOthersEditing     = 2091110
	minutesWordReplaceWordsNotFound     = 2091013
	minutesWordReplaceASRQuotaNotEnough = 2091008
)

const minutesWordReplaceDoNotRetrySucceeded = "Do not reprocess words that already succeeded."

const minutesWordReplaceNotFoundHint = "Report the unmatched source_word values and confirm their exact spelling, case and spacing with the user, then retry only those words"

type transcriptWordReplace struct {
	SourceWord string  `json:"source_word"`
	TargetWord *string `json:"target_word"`
}

// MinutesWordReplace batch-replaces words in a minute's transcript.
var MinutesWordReplace = common.Shortcut{
	Service:     "minutes",
	Command:     "+word-replace",
	Description: "Batch replace words in a minute's transcript",
	Risk:        "write",
	Scopes:      []string{"minutes:minutes:update"},
	AuthTypes:   []string{"user"},
	HasFormat:   true,
	Flags: []common.Flag{
		{Name: "minute-token", Desc: "minute token", Required: true},
		{
			Name:     "replace-words",
			Desc:     `JSON array of replacements, e.g. [{"source_word":"old","target_word":"new"}]`,
			Required: true,
			Input:    []string{common.File, common.Stdin},
		},
	},
	Validate: func(ctx context.Context, runtime *common.RuntimeContext) error {
		minuteToken := strings.TrimSpace(runtime.Str("minute-token"))
		if minuteToken == "" {
			return errs.NewValidationError(errs.SubtypeInvalidArgument, "--minute-token is required").WithParam("--minute-token")
		}
		if err := validate.ResourceName(minuteToken, "--minute-token"); err != nil {
			return errs.NewValidationError(errs.SubtypeInvalidArgument, "%s", err).WithParam("--minute-token")
		}
		if _, err := parseReplaceWords(runtime.Str("replace-words")); err != nil {
			return err
		}
		return nil
	},
	DryRun: func(ctx context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
		minuteToken := strings.TrimSpace(runtime.Str("minute-token"))
		replaceWords, _ := parseReplaceWords(runtime.Str("replace-words"))
		return common.NewDryRunAPI().
			PUT(fmt.Sprintf("/open-apis/minutes/v1/minutes/%s/transcript/word", validate.EncodePathSegment(minuteToken))).
			Body(map[string]interface{}{
				"minute_token":  minuteToken,
				"replace_words": replaceWords,
			})
	},
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		minuteToken := strings.TrimSpace(runtime.Str("minute-token"))
		replaceWords, err := parseReplaceWords(runtime.Str("replace-words"))
		if err != nil {
			return err
		}

		body := map[string]interface{}{
			"minute_token":  minuteToken,
			"replace_words": replaceWords,
		}

		data, err := runtime.CallAPITyped(http.MethodPut,
			fmt.Sprintf("/open-apis/minutes/v1/minutes/%s/transcript/word", validate.EncodePathSegment(minuteToken)),
			nil, body)
		if err != nil {
			return minutesWordReplaceError(err, minuteToken)
		}

		return emitWordReplaceResult(runtime, minuteToken, replaceWords, data)
	},
}

func parseReplaceWords(raw string) ([]map[string]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errs.NewValidationError(errs.SubtypeInvalidArgument, "--replace-words: value is required").WithParam("--replace-words")
	}

	var items []transcriptWordReplace
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, errs.NewValidationError(errs.SubtypeInvalidArgument, "--replace-words: must be a JSON array of {source_word,target_word} objects: %v", err).WithParam("--replace-words").WithCause(err)
	}
	if len(items) == 0 {
		return nil, errs.NewValidationError(errs.SubtypeInvalidArgument, "--replace-words: must include at least one replacement").WithParam("--replace-words")
	}

	replaceWords := make([]map[string]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for i, item := range items {
		sourceWord := item.SourceWord
		if strings.TrimSpace(sourceWord) == "" {
			return nil, errs.NewValidationError(errs.SubtypeInvalidArgument, "--replace-words: item %d: source_word is required", i).WithParam("--replace-words")
		}
		if _, exists := seen[sourceWord]; exists {
			return nil, errs.NewValidationError(errs.SubtypeInvalidArgument, "--replace-words: duplicate source_word %q", sourceWord).WithParam("--replace-words")
		}
		if item.TargetWord == nil {
			return nil, errs.NewValidationError(errs.SubtypeInvalidArgument, "--replace-words: item %d: target_word is required (use an empty string for deletion)", i).WithParam("--replace-words")
		}
		seen[sourceWord] = struct{}{}
		replaceWords = append(replaceWords, map[string]string{
			"source_word": sourceWord,
			"target_word": *item.TargetWord,
		})
	}
	return replaceWords, nil
}

func emitWordReplaceResult(runtime *common.RuntimeContext, minuteToken string, replaceWords []map[string]string, data map[string]interface{}) error {
	counts, hasCounts := parseReplaceWordCounts(data)
	var unknown []string
	for _, item := range replaceWords {
		if _, found := counts[item["source_word"]]; !found {
			unknown = append(unknown, item["source_word"])
		}
	}
	if hasCounts && len(counts) > 0 && len(unknown) > 0 {
		return runtime.OutPartialFailure(map[string]any{
			"minute_token":        minuteToken,
			"replace_word_counts": counts,
			"unknown":             unknown,
			"message":             "The server omitted outcomes for some requested words. Check the transcript before deciding whether to retry. " + minutesWordReplaceDoNotRetrySucceeded,
		}, nil)
	}
	if !hasCounts || len(unknown) > 0 {
		return errs.NewValidationError(errs.SubtypeFailedPrecondition,
			"Replace request returned success, but the server did not include replace_word_counts, so per-word outcomes cannot be reported.").
			WithHint("Check the transcript with `minutes +detail --minute-tokens %s --transcript` before retrying. %s",
				minuteToken, minutesWordReplaceDoNotRetrySucceeded)
	}

	succeeded, failed := splitWordReplaceResults(replaceWords, counts)
	if len(succeeded) == 0 {
		return minutesWordReplaceAllFailed(minuteToken, failed)
	}

	result := map[string]interface{}{
		"minute_token":        minuteToken,
		"message":             formatWordReplaceMessage(succeeded, failed),
		"succeeded":           succeeded,
		"failed":              failed,
		"replace_word_counts": counts,
	}
	if len(failed) > 0 {
		return runtime.OutPartialFailure(result, nil)
	}
	runtime.OutFormat(result, nil, nil)
	return nil
}

func formatWordReplaceMessage(succeeded, failed []string) string {
	var b strings.Builder
	b.WriteString("Succeeded: ")
	if len(succeeded) == 0 {
		b.WriteString("none")
	} else {
		b.WriteString(strings.Join(succeeded, ", "))
	}
	b.WriteString("; Failed: ")
	if len(failed) == 0 {
		b.WriteString("none")
	} else {
		b.WriteString(strings.Join(failed, ", "))
	}
	b.WriteString(". ")
	b.WriteString(minutesWordReplaceDoNotRetrySucceeded)
	return b.String()
}

// minutesWordReplaceAllFailed reports an all-zero replace_word_counts response.
// The call itself succeeded, so there is no upstream code to carry here.
func minutesWordReplaceAllFailed(minuteToken string, failed []string) error {
	msg := fmt.Sprintf("None of the source words were found in minute %q transcript; nothing was replaced.", minuteToken)
	if len(failed) > 0 {
		msg = fmt.Sprintf("Succeeded: none; Failed: %s. %s", strings.Join(failed, ", "), msg)
	}
	return errs.NewAPIError(errs.SubtypeNotFound, "%s", msg).
		WithHint("%s", minutesWordReplaceNotFoundHint)
}

func parseReplaceWordCounts(data map[string]interface{}) (map[string]int64, bool) {
	var response struct {
		Counts *[]struct {
			Source string          `json:"source_word"`
			Count  json.RawMessage `json:"replace_count"`
		} `json:"replace_word_counts"`
	}
	encoded, err := json.Marshal(data)
	if err != nil || json.Unmarshal(encoded, &response) != nil || response.Counts == nil {
		return nil, false
	}
	counts := make(map[string]int64, len(*response.Counts))
	for _, item := range *response.Counts {
		if strings.TrimSpace(item.Source) == "" {
			return nil, false
		}
		if _, duplicate := counts[item.Source]; duplicate {
			return nil, false
		}
		raw := string(item.Count)
		if strings.HasPrefix(raw, "\"") {
			if json.Unmarshal(item.Count, &raw) != nil {
				return nil, false
			}
		}
		count, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || count < 0 {
			return nil, false
		}
		counts[item.Source] = count
	}
	return counts, true
}

func splitWordReplaceResults(replaceWords []map[string]string, counts map[string]int64) (succeeded, failed []string) {
	succeeded = make([]string, 0, len(replaceWords))
	failed = make([]string, 0)
	for _, item := range replaceWords {
		source := item["source_word"]
		count, ok := counts[source]
		if !ok || count <= 0 {
			failed = append(failed, source)
			continue
		}
		succeeded = append(succeeded, source)
	}
	return succeeded, failed
}

func minutesWordReplaceError(err error, minuteToken string) error {
	p, ok := errs.ProblemOf(err)
	if !ok {
		return err
	}

	switch p.Code {
	case minutesWordReplaceNoEditPermission:
		p.Subtype = errs.SubtypePermissionDenied
		p.Message = fmt.Sprintf("No edit permission for minute %q: cannot replace transcript words.", minuteToken)
		p.Hint = fmt.Sprintf("Ask the user before running: minutes +apply-permission --minute-token %s --perm edit", minuteToken)
	case minutesWordReplaceOthersEditing:
		p.Subtype = errs.SubtypeConflict
		p.Message = fmt.Sprintf("Minute %q transcript is being edited by someone else.", minuteToken)
		p.Hint = "Wait until the other editor finishes, then retry"
	case minutesWordReplaceWordsNotFound:
		p.Subtype = errs.SubtypeNotFound
		p.Message = fmt.Sprintf("None of the source words were found in minute %q transcript; nothing was replaced.", minuteToken)
		p.Hint = minutesWordReplaceNotFoundHint
	case minutesWordReplaceASRQuotaNotEnough:
		p.Subtype = errs.SubtypeQuotaExceeded
		p.Message = fmt.Sprintf("ASR/AI quota not enough: cannot replace transcript words on minute %q.", minuteToken)
		p.Hint = minutesASRQuotaNotEnoughHint
	}

	return err
}
