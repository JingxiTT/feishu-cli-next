# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Bounded historical native revision limitation, without inventing a CAS API."""
import copy
from urllib.parse import parse_qs, urlencode, urlsplit, urlunsplit

from scenarios import docs as native

UNSUPPORTED_HISTORY = "unsupported strategy: historical native Docs revision is outside this fixture"


def reject_unmodeled_history(scenario, method, uri, body_bytes, headers):
    """Validate the current equivalent on a disposable state before classifying.

    The native probe in docs-workspace-native-cas-20260909.json observed historical
    block-list reads and successful stale PATCH, not a revision-conflict guard.
    We do not model either history selection or server edit transformation here.
    Call under the scenario lock; neither validation nor rejection touches its
    state, mutations or read observations. Docs-AI revision handling is unchanged.
    """
    parsed = urlsplit(uri)
    prefix = native.DOCX_PATH + "/blocks/"
    if not ((method == "GET" and parsed.path == native.DOCX_PATH + "/blocks") or
            (method == "PATCH" and parsed.path.startswith(prefix) and
             "/" not in parsed.path[len(prefix):])):
        return
    query = parse_qs(parsed.query, keep_blank_values=True)
    values = query.get("document_revision_id", ["-1"])
    if any(len(values) != 1 for values in query.values()):
        return  # The original dispatcher retains duplicate-parameter rejection.
    revision = native.integer(values[0], "revision")
    if revision == -1 or revision >= scenario.state["revision_id"]:
        return
    # Validate every other native route/body/ID constraint, including fields
    # checked after revision parsing, before calling this a fixture limitation.
    validation = native.Scenario()
    validation.state = copy.deepcopy(scenario.state)
    query["document_revision_id"] = [str(validation.state["revision_id"])]
    current_uri = urlunsplit(parsed._replace(query=urlencode(query, doseq=True)))
    validation.dispatch(method, current_uri, body_bytes, headers)
    raise ValueError(UNSUPPORTED_HISTORY)
