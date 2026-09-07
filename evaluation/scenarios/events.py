"""Two native event deliveries over a private, bounded Unix IPC fixture.

No remote WebSocket, mutation endpoint, persistent replay service, or model call.
"""
import copy
import ctypes
import json
import os
import shlex
from pathlib import Path
import socket
import struct
import sys
import tempfile
import threading
import time
from urllib.parse import parse_qs, urlsplit

from scenario_protocol import Reply, ScenarioBase

APP_ID = "synthetic-evaluation-events"
KEY = "im.chat.updated_v1"
NAME = "events-two-native-v1"
MAX_CONNECTIONS = 16
TERMINATION_SECONDS = 5.0


def _payload(event_id, text, timestamp):
    return {"schema": "2.0", "header": {"event_id": event_id, "event_type": KEY,
            "create_time": timestamp, "app_id": APP_ID, "tenant_key": "fixture-tenant"},
            "event": {"chat_id": "oc_fixture_events", "after_change": {"name": text}}}


PAYLOADS = [_payload("e1", "你好", "1700000000001"), _payload("e2", "", "1700000000002")]
EXPECTED = {"answer": {"events": [{"id": "e1", "text": "你好"}, {"id": "e2", "text": ""}]},
            "payloads": PAYLOADS}
TASK = {
    "id": "events-dev-01", "section": "events",
    "prompt": "Consume the two synthetic im.chat.updated_v1 events available through the configured CLI event bus, then stop the consumer cleanly after the second event. Save the complete received JSON objects as events.ndjson in the working directory. Preserve both event IDs and the new chat names exactly, including Unicode and the empty second name. Return JSON with an events array of {id,text} values. This is a read-only local fixture task; leave the fixture bus running and do not change resources.",
    "fixture": {"scenario": NAME, "answer_schema": {"type": "object", "required": ["events"],
        "additionalProperties": False, "properties": {"events": {"type": "array", "items": {
        "type": "object", "required": ["id", "text"], "additionalProperties": False,
        "properties": {"id": {"type": "string"}, "text": {"type": "string"}}}}}}},
    "expected": EXPECTED,
}


def _peer_pid(conn):
    if sys.platform == "darwin":
        # Darwin SDK sys/un.h: SOL_LOCAL=0, LOCAL_PEERPID=0x002.
        return conn.getsockopt(0, 2)
    if sys.platform.startswith("linux"):
        return struct.unpack("3i", conn.getsockopt(socket.SOL_SOCKET, socket.SO_PEERCRED, 12))[0]
    raise OSError("event fixture supports macOS/Linux Unix peer credentials only")


def _executable(pid):
    if sys.platform == "darwin":
        lib = ctypes.CDLL("/usr/lib/libproc.dylib", use_errno=True)
        buffer = ctypes.create_string_buffer(4096)
        if lib.proc_pidpath(pid, buffer, len(buffer)) > 0:
            return os.fsdecode(buffer.value)
        return ""
    try:
        return os.readlink(f"/proc/{pid}/exe")
    except OSError:
        return ""


def _alive(pid):
    try:
        os.kill(pid, 0)
        return True
    except ProcessLookupError:
        return False
    except PermissionError:
        return True


class Scenario(ScenarioBase):
    def __init__(self):
        self._lock = threading.Lock()
        self._stop = threading.Event()
        self._connections = set()
        self._workers = []
        self._records = []
        self._rejected = []
        self._directory = None
        self._listener = None
        self._thread = None
        self._closed = False
        self._active_before_close = None
        self.selected_cli = None
        self._executions = []

    def bind_cli(self, path):
        """Bind the runner-selected executable before exposing the listener."""
        if self._directory is not None:
            raise RuntimeError("bind CLI before setup")
        self.selected_cli = str(Path(path).resolve(strict=True))

    def setup(self, workspace):
        if sys.platform != "darwin" and not sys.platform.startswith("linux"):
            raise RuntimeError("events scenario requires macOS/Linux Unix sockets")
        if self._directory is not None:
            raise RuntimeError("scenario is already set up")
        self.workspace = Path(workspace)
        if self.selected_cli is not None:
            self.selected_cli = str(Path(self.selected_cli).resolve(strict=True))
        self._directory = tempfile.TemporaryDirectory(prefix="eval-events-", dir="/tmp")
        self._address = Path(self._directory.name) / "events" / APP_ID / "bus.sock"
        self._address.parent.mkdir(parents=True, mode=0o700)
        self._listener = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        self._listener.bind(str(self._address))
        self._listener.listen()
        self._listener.settimeout(0.1)
        self._thread = threading.Thread(target=self._accept, daemon=True)
        self._thread.start()

    def environment(self):
        if self._directory is None:
            raise RuntimeError("setup must precede environment")
        return {"LARKSUITE_CLI_CONFIG_DIR": self._directory.name, "LARKSUITE_CLI_APP_ID": APP_ID,
                "LARKSUITE_CLI_BRAND": "feishu", "LARKSUITE_CLI_DEFAULT_AS": "bot", "TZ": "UTC"}

    def unix_sockets(self):
        return [str(self._address)] if self._directory is not None else []

    def dispatch(self, method, uri, body_bytes, headers):
        parsed = urlsplit(uri)
        identity = headers.get("X-Lark-Proxy-Identity")
        if method != "GET" or body_bytes or identity not in ("user", "bot") or parsed.scheme or parsed.netloc or parsed.fragment:
            raise ValueError("events fixture permits only signed identity/app reads")
        query = parse_qs(parsed.query, keep_blank_values=True)
        if parsed.path == "/open-apis/authen/v1/user_info" and not query:
            return Reply("identity", json={"code": 0, "data": {"open_id": "fixture_self", "name": "Synthetic Fixture", "tenant_key": "fixture-tenant"}})
        if identity != "bot":
            raise ValueError("app preflight requires bot identity")
        if parsed.path == f"/open-apis/application/v6/applications/{APP_ID}/app_versions":
            if set(query) - {"lang", "page_size", "page_token"} or query.get("lang", ["zh_cn"]) not in (["zh_cn"], ["en_us"]):
                raise ValueError("invalid version query")
            size = query.get("page_size", ["2"])
            if len(size) != 1 or not size[0].isdigit() or not 1 <= int(size[0]) <= 100 or query.get("page_token", [""]) != [""]:
                raise ValueError("invalid version pagination")
            return Reply("event_preflight", json={"code": 0, "data": {"items": []}}, proof={"app_id": APP_ID})
        if parsed.path == "/open-apis/event/v1/connection" and not query:
            return Reply("event_connections", json={"code": 0, "data": {"online_instance_cnt": 0}}, proof={"app_id": APP_ID})
        raise ValueError("unsupported event fixture read")

    def observe_execution(self, item):
        """Consume trusted completed-command evidence; retain no raw command/output.

        A simple foreground invocation's shell status is its CLI status. Pipelines,
        compound shells and substitutions cannot establish the child's exit here.
        """
        if item.get("type") != "command_execution" or self.selected_cli is None:
            return
        command = item.get("command", "")
        try:
            tokens = shlex.split(command)
            if len(tokens) == 3 and Path(tokens[0]).name in {"sh", "bash", "zsh"} and tokens[1] in {"-c", "-lc"}:
                command = tokens[2]
            if any(char in command for char in ("$", "`", "\n")):
                return
            lexer = shlex.shlex(command, posix=True, punctuation_chars=";&|<>")
            lexer.whitespace_split = True
            tokens = list(lexer)
            if any(token in {";", "&", "&&", "|", "||", "<", "<<", "<<<", ">>"} for token in tokens):
                return
            arguments = []
            cursor = 0
            while cursor < len(tokens):
                if tokens[cursor] == ">":
                    if cursor + 1 >= len(tokens):
                        return
                    if arguments and arguments[-1] in {"1", "2"}:
                        arguments.pop()
                    cursor += 2
                    continue
                arguments.append(tokens[cursor])
                cursor += 1
            if len(arguments) < 4:
                return
            executable = Path(arguments[0])
            if executable.is_absolute():
                candidate = executable
            elif arguments[0] in {"lark-cli", "feishu-cli-next"}:
                if not hasattr(self, "workspace"):
                    return
                candidate = self.workspace / "bin" / arguments[0]
            elif executable.name != arguments[0] and hasattr(self, "workspace"):
                candidate = self.workspace / executable
            else:
                return
            if (not candidate.is_file() or str(candidate.resolve()) != self.selected_cli or
                    arguments[1:3] != ["event", "consume"] or KEY not in arguments[3:]
                    or any(flag in arguments for flag in ("--help", "-h", "--dry-run"))
                    or type(item.get("exit_code")) is not int):
                return
        except (ValueError, TypeError, OSError):
            return
        with self._lock:
            self._executions.append({"simple_selected_cli_consume": True, "exit_code": item["exit_code"]})

    def _accept(self):
        accepted = 0
        while not self._stop.is_set():
            try:
                conn, _ = self._listener.accept()
            except socket.timeout:
                continue
            except OSError:
                return
            accepted += 1
            if accepted > MAX_CONNECTIONS:
                conn.close()
                with self._lock:
                    self._rejected.append("connection_limit")
                continue
            with self._lock:
                self._connections.add(conn)
            worker = threading.Thread(target=self._serve, args=(conn,), daemon=True)
            self._workers.append(worker)
            worker.start()

    def _serve(self, conn):
        episode = None
        ended = None
        try:
            peer_pid = _peer_pid(conn)
            executable = _executable(peer_pid)
            conn.settimeout(10)
            with conn.makefile("rb") as reader:
                for _ in range(8):
                    line = reader.readline(65537)
                    if not line:
                        ended = "eof"
                        break
                    if len(line) > 65536:
                        raise ValueError("oversized frame")
                    message = json.loads(line)
                    if not isinstance(message, dict):
                        raise ValueError("object frame required")
                    kind = message.get("type")
                    def send(value):
                        conn.sendall((json.dumps(value, ensure_ascii=False, separators=(",", ":")) + "\n").encode())
                    if kind == "status_query" and episode is None:
                        with self._lock:
                            consumers = [{"pid": item["peer_pid"], "event_key": KEY}
                                         for item in self._records if not item["connection_closed"]]
                        send({"type": "status_response", "pid": os.getpid(), "uptime_sec": 1,
                              "active_conns": len(consumers), "consumers": consumers})
                        return
                    if kind == "hello" and episode is None:
                        if (type(message.get("pid")) is not int or message["pid"] != peer_pid or
                                message.get("event_key") != KEY or message.get("event_types") != [KEY]):
                            raise ValueError("wrong consumer identity or key")
                        episode = {"peer_pid": peer_pid, "executable": executable,
                                   "delivered_ids": [], "connection_closed": False,
                                   "terminated": False, "forced_disconnect": False}
                        with self._lock:
                            self._records.append(episode)
                        send({"type": "hello_ack", "bus_version": "fixture", "first_for_key": False,
                              "capabilities": ["canonical_metadata_v1"]})
                        for seq, payload in enumerate(PAYLOADS, 1):
                            send({"type": "event", "event_type": KEY, "event_id": payload["header"]["event_id"],
                                  "source_time": payload["header"]["create_time"], "app_id": APP_ID,
                                  "tenant_key": "fixture-tenant", "observed_at": "2023-11-14T22:13:21Z",
                                  "seq": seq, "payload": payload})
                            with self._lock:
                                episode["delivered_ids"].append(payload["header"]["event_id"])
                        episode["_delivered_at"] = time.monotonic()
                    elif kind == "pre_shutdown_check" and episode is not None and message.get("event_key") == KEY:
                        send({"type": "pre_shutdown_ack", "last_for_key": True})
                    elif kind == "bye" and episode is not None:
                        ended = "bye"
                        break
                    else:
                        raise ValueError("unexpected protocol frame")
        except (OSError, ValueError, TypeError):
            with self._lock:
                self._rejected.append("invalid_or_interrupted_connection")
        finally:
            conn.close()
            with self._lock:
                self._connections.discard(conn)
                if episode is not None:
                    episode["connection_closed"] = ended in ("eof", "bye")
                    episode["forced_disconnect"] = self._stop.is_set() and ended is None
            if episode is not None and ended is not None:
                deadline = episode.get("_delivered_at", time.monotonic()) + TERMINATION_SECONDS
                while _alive(episode["peer_pid"]) and time.monotonic() < deadline:
                    time.sleep(0.01)
                elapsed = time.monotonic() - episode.get("_delivered_at", time.monotonic())
                with self._lock:
                    episode["terminated"] = not _alive(episode["peer_pid"]) and elapsed <= TERMINATION_SECONDS
                    episode["termination_seconds"] = round(elapsed, 3)

    def close(self):
        if self._closed:
            return
        self._closed = True
        # Allow a just-exited subprocess's EOF to be observed before teardown.
        deadline = time.monotonic() + 0.25
        while time.monotonic() < deadline:
            with self._lock:
                if not self._connections:
                    break
            time.sleep(0.01)
        with self._lock:
            self._active_before_close = len(self._connections)
            connections = list(self._connections)
        self._stop.set()
        if self._listener:
            self._listener.close()
        for conn in connections:
            try:
                conn.shutdown(socket.SHUT_RDWR)
            except OSError:
                pass
            conn.close()
        if self._thread:
            self._thread.join(1)
        deadline = time.monotonic() + TERMINATION_SECONDS + 0.1
        for worker in self._workers:
            worker.join(max(0, deadline - time.monotonic()))
        if self._directory:
            self._directory.cleanup()

    def snapshot(self):
        with self._lock:
            return {"scenario": NAME, "episodes": [{k: copy.deepcopy(v) for k, v in episode.items()
                    if not k.startswith("_")} for episode in self._records],
                    "rejected": list(self._rejected), "active_before_teardown": self._active_before_close,
                    "unix_only": True, "selected_cli": self.selected_cli,
                    "executions": copy.deepcopy(self._executions)}


def verify(workspace, answer, receipt, expected):
    try:
        if not isinstance(receipt, dict) or answer != expected["answer"] or receipt.get("scenario") != NAME:
            return False
        requests = receipt.get("requests", [])
        if not requests or any(r.get("route") not in {"identity", "event_preflight", "event_connections"}
                               or r.get("status") != 200 for r in requests):
            return False
        if not any(r.get("route") == "event_preflight" for r in requests):
            return False
        state = receipt.get("state", {})
        if state.get("scenario") != NAME or state.get("rejected") or type(state.get("active_before_teardown")) is not int or state["active_before_teardown"] != 0:
            return False
        selected_cli = state.get("selected_cli")
        if not isinstance(selected_cli, str) or not selected_cli:
            return False
        episodes = state.get("episodes", [])
        if not episodes or any(e.get("delivered_ids") != ["e1", "e2"] or e.get("connection_closed") is not True
                               or e.get("terminated") is not True or e.get("forced_disconnect")
                               or e.get("executable") != selected_cli or type(e.get("peer_pid")) is not int
                               or e["peer_pid"] <= 0 or type(e.get("termination_seconds")) not in (int, float)
                               or not 0 <= e["termination_seconds"] <= TERMINATION_SECONDS for e in episodes):
            return False
        executions = state.get("executions", [])
        if (len(executions) != len(episodes) or any(e.get("simple_selected_cli_consume") is not True
                or type(e.get("exit_code")) is not int or e["exit_code"] != 0 for e in executions)):
            return False
        artifact = Path(workspace) / "events.ndjson"
        if artifact.is_symlink() or not artifact.is_file() or artifact.stat().st_size > 65536:
            return False
        lines = artifact.read_text(encoding="utf-8").splitlines()
        return len(lines) == 2 and [json.loads(line) for line in lines] == expected["payloads"]
    except (OSError, ValueError, TypeError, KeyError, AttributeError):
        return False
