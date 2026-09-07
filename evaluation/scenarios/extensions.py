"""Public-host fixture with independently observed, bounded command execution.

The runner supplies selected_cli and calls observe_execution with real completed
Codex items. This is controlled execution evidence, not a filesystem-read audit.
"""
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import shlex
import subprocess
import tempfile

from scenario_protocol import ScenarioBase


ROOT = Path(__file__).resolve().parents[2]
TASK = {
    "id": "extensions-dev-01",
    "section": "extensions",
    "prompt": "Run the synthetic-extension-host integration command with the isolated configuration supplied by the harness. Capture its complete JSON stdout in host-result.json in your workspace and report the returned value, preserving its JSON type. Do not change configuration or call a live service.",
    "fixture": {"scenario": "extensions-dev-01", "answer_schema": {
        "type": "object", "properties": {"value": {"type": "integer"}},
        "required": ["value"], "additionalProperties": False}},
    "expected": {"value": 0},
}


def build_host(source_root, output_path, next_mode):
    """Build the reviewed external wrapper from a caller-selected clean archive.

    Returns the executable Path (.exe on Windows). Does not modify the archive,
    install globally or inherit real credential/configuration environments.
    """
    source = Path(source_root).resolve(strict=True)
    output = Path(output_path).resolve()
    if source == output or source in output.parents:
        raise ValueError("host output must be outside the clean source archive")
    if type(next_mode) is not bool:
        raise ValueError("next_mode must be boolean")
    if os.name == "nt" and output.suffix.lower() != ".exe":
        output = output.with_name(output.name + ".exe")
    contract = json.loads((ROOT / "evaluation/contracts/extensions.json").read_text())
    template = contract["host_build"]["main_go_template"]
    marker = "// CANDIDATE_AGENT_OPTION"
    if template.count(marker) != 1:
        raise ValueError("reviewed host template must have exactly one mode marker")
    main = template.replace(marker, "options = append(options, cmd.WithAgentMode())" if next_mode else "")
    module = (source / "go.mod").read_text()
    if module.count("module github.com/larksuite/cli") != 1:
        raise ValueError("source is not the reviewed CLI module")
    module = module.replace("module github.com/larksuite/cli", "module extensionevalhost", 1)
    module += "\nrequire github.com/larksuite/cli v0.0.0\nreplace github.com/larksuite/cli => " + json.dumps(source.as_posix(), ensure_ascii=False) + "\n"
    spec = importlib.util.spec_from_file_location("extension_build_isolation", ROOT / "scripts/next_test.py")
    isolation = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(isolation)
    with tempfile.TemporaryDirectory(prefix="next-tests-") as directory:
        work = Path(directory) / "module"
        work.mkdir()
        (work / "go.mod").write_text(module)
        (work / "go.sum").write_bytes((source / "go.sum").read_bytes())
        (work / "main.go").write_text(main)
        output.parent.mkdir(parents=True, exist_ok=True)
        env = isolation.isolated_env(directory, os.environ, "http://127.0.0.1:1")
        completed = subprocess.run(["go", "build", "-mod=mod", "-o", str(output), "."], cwd=work,
                                   env=env, capture_output=True, text=True, timeout=300, check=False)
        if completed.returncode:
            raise RuntimeError("external host build failed: " + completed.stderr[-4000:])
    return output


class Scenario(ScenarioBase):
    def setup(self, workspace):
        self.workspace = Path(workspace).resolve()
        self.config = self.workspace / ".extension-config"
        self.config.mkdir(exist_ok=False)
        self.executions = []

    def environment(self):
        return {"LARKSUITE_CLI_CONFIG_DIR": str(self.config), "LARKSUITE_CLI_REMOTE_META": "off",
                "LARKSUITE_CLI_NO_UPDATE_NOTIFIER": "1", "LARKSUITE_CLI_NO_SKILLS_NOTIFIER": "1"}

    def dispatch(self, method, uri, body_bytes, headers):
        raise ValueError("the public host fixture performs no API requests")

    def snapshot(self):
        try:
            unchanged = self.config.is_dir() and not self.config.is_symlink() and not any(self.config.iterdir())
        except OSError:
            unchanged = False
        return {"config_unchanged": unchanged, "host_executions": list(self.executions)}

    def observe_execution(self, item):
        """Accept only an actual simple selected-host invocation, never shell prose."""
        try:
            if item.get("type") != "command_execution" or type(item.get("exit_code")) is not int or item["exit_code"] != 0:
                return
            command = item["command"]
            if not isinstance(command, str):
                return
            outer = shlex.split(command)
            if len(outer) == 3 and Path(outer[0]).name in ("sh", "bash", "zsh") and outer[1] in ("-c", "-lc"):
                command = outer[2]
            lexer = shlex.shlex(command, posix=True, punctuation_chars=";&|<>()")
            lexer.whitespace_split = True
            lexer.commenters = ""
            words = list(lexer)
            selected = Path(self.selected_cli).resolve()
            if not words:
                return
            executable = words.pop(0)
            absolute_match = Path(executable).is_absolute() and Path(executable).resolve() == selected
            if executable != selected.name and not absolute_match:
                alias = self.workspace / "bin" / executable
                if Path(executable).name != executable or not alias.is_file() or alias.resolve() != selected:
                    return
            redirect = False
            if ">" in words:
                position = words.index(">")
                if position != len(words) - 2 or Path(words[-1]).is_absolute() and Path(words[-1]) != self.workspace / "host-result.json":
                    return
                target = (self.workspace / words[-1]).resolve()
                if target != self.workspace / "host-result.json":
                    return
                words = words[:position]
                redirect = True
            for flag, value in (("--as", "bot"), ("--format", "json")):
                if flag in words:
                    position = words.index(flag)
                    if words[position:position + 2] != [flag, value]:
                        return
                    del words[position:position + 2]
                elif flag + "=" + value in words:
                    words.remove(flag + "=" + value)
            if words != ["im", "+fixture-host"]:
                return
            observed = item.get("aggregated_output")
            if not isinstance(observed, str):
                return
            if redirect:
                if observed:
                    return
                raw = _artifact_bytes(self.workspace)
            else:
                raw = observed.encode("utf-8")
            if _envelope(raw):
                self.executions.append({"command_path": "im +fixture-host", "exit_code": 0,
                                        "capture": "redirect" if redirect else "observed_output",
                                        "sha256": _digest(raw)})
        except (KeyError, TypeError, ValueError, OSError, AttributeError):
            return


def _zero(value):
    return type(value) is int and value == 0


def _envelope(raw):
    value = json.loads(raw)
    return (isinstance(value, dict) and value.get("ok") is True and value.get("identity") == "bot"
            and isinstance(value.get("data"), dict) and _zero(value["data"].get("value")))


def _artifact_bytes(workspace):
    artifact = Path(workspace) / "host-result.json"
    if artifact.is_symlink() or not artifact.is_file() or artifact.stat().st_size > 65536:
        raise ValueError("missing or invalid host artifact")
    return artifact.read_bytes()


def _digest(raw):
    return hashlib.sha256(raw.rstrip(b"\r\n")).hexdigest()


def verify(workspace, answer, receipt, expected):
    """Require observed selected-host execution, unchanged config and actual bytes."""
    try:
        if not isinstance(answer, dict) or not _zero(answer.get("value")) or not _zero(expected.get("value")):
            return False
        if receipt.get("scenario") != TASK["id"] or receipt.get("requests") != []:
            return False
        state = receipt["state"]
        if state.get("config_unchanged") is not True:
            return False
        raw = _artifact_bytes(workspace)
        return _envelope(raw) and any(
            proof.get("command_path") == "im +fixture-host" and type(proof.get("exit_code")) is int
            and proof["exit_code"] == 0 and proof.get("capture") in ("redirect", "observed_output")
            and proof.get("sha256") == _digest(raw) for proof in state["host_executions"])
    except (KeyError, TypeError, ValueError, OSError, AttributeError):
        return False
