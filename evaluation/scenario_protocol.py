# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Minimal interface for stateful/artifact/offline development fixtures."""
from dataclasses import dataclass, field


@dataclass
class Reply:
    route: str
    json: object = None
    body: bytes | None = None
    status: int = 200
    content_type: str = "application/json"
    headers: dict = field(default_factory=dict)
    proof: dict = field(default_factory=dict)


class ScenarioBase:
    def setup(self, workspace):
        """Stage public local inputs/start a local listener; never write answers."""

    def environment(self):
        return {}

    def unix_sockets(self):
        return []

    def close(self):
        pass

    def snapshot(self):
        return {}
