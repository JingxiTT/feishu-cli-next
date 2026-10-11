// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package task

import (
	"math"
	"strconv"
	"strings"

	"github.com/larksuite/cli/errs"
)

func parseReminderMinutes(input string) (int, error) {
	value, multiplier := input, 1
	for suffix, factor := range map[string]int{"m": 1, "h": 60, "d": 1440} {
		if strings.HasSuffix(input, suffix) {
			value, multiplier = strings.TrimSuffix(input, suffix), factor
			break
		}
	}
	amount, err := strconv.Atoi(value)
	if err != nil {
		return 0, errs.NewValidationError(errs.SubtypeInvalidArgument, "invalid reminder duration").WithParam("--set").WithCause(err)
	}
	if amount < 0 || amount > math.MaxInt/multiplier {
		return 0, errs.NewValidationError(errs.SubtypeInvalidArgument, "reminder duration must be a non-negative number of minutes without overflow").WithParam("--set")
	}
	return amount * multiplier, nil
}
