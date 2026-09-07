"""Portable shebang dispatch for a controlled executable fixture.

The actual adapter/verifier entry point and fixture run in separate real Python
processes. Only launching the named .py fixture is translated to sys.executable;
this replaces POSIX shebang handling, not adapter behavior or provider events.
"""
import argparse
from pathlib import Path
import runpy
import subprocess
import sys

parser = argparse.ArgumentParser()
parser.add_argument('--entry', required=True)
parser.add_argument('--fixture', required=True)
args, remaining = parser.parse_known_args()
if remaining and remaining[0] == '--':
    remaining = remaining[1:]
fixture = Path(args.fixture).resolve()
original_run, original_popen = subprocess.run, subprocess.Popen


def invocation(command):
    if isinstance(command, (list, tuple)) and command and Path(command[0]).resolve() == fixture:
        return [sys.executable, *command]
    return command


subprocess.run = lambda command, *a, **kw: original_run(invocation(command), *a, **kw)
subprocess.Popen = lambda command, *a, **kw: original_popen(invocation(command), *a, **kw)
sys.path.insert(0, str(Path(args.entry).resolve().parent))
sys.argv = [args.entry, *remaining]
runpy.run_path(args.entry, run_name='__main__')
