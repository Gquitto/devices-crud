#!/usr/bin/env python3
"""PreToolUse hook: block the agent from creating, editing or deleting tests.

In this project the user writes every test; the agent only writes production
code. Exit code 2 blocks the tool call and shows stderr to the agent.
"""
import json
import re
import sys

PROTECTED = re.compile(r"(_test\.go$|(^|/)testdata(/|$))")
TEST_REF = re.compile(r"_test\.go|testdata/")
# Heuristic: shell operations that could modify a referenced test file.
SHELL_WRITE = re.compile(
    r"\b(rm|mv|cp|tee|truncate|touch|install|dd|patch)\b"
    r"|sed\s+-i"
    r"|>{1,2}\s*\S*(_test\.go|testdata/)"
)


def block(reason: str) -> None:
    print(
        f"BLOCKED: {reason}\n"
        "Tests are owned by the user (see CLAUDE.md). Do not create, edit or "
        "delete *_test.go files or testdata/. Change production code instead, "
        "or ask the user if a test looks wrong.",
        file=sys.stderr,
    )
    sys.exit(2)


def main() -> None:
    try:
        payload = json.load(sys.stdin)
    except json.JSONDecodeError:
        return

    tool = payload.get("tool_name", "")
    tool_input = payload.get("tool_input") or {}

    if tool in {"Edit", "Write", "MultiEdit", "NotebookEdit"}:
        path = tool_input.get("file_path") or tool_input.get("notebook_path") or ""
        if PROTECTED.search(path):
            block(f"{tool} on protected test path '{path}'")

    elif tool == "Bash":
        cmd = tool_input.get("command", "")
        if TEST_REF.search(cmd) and SHELL_WRITE.search(cmd):
            block(f"shell command may modify test files: {cmd!r}")


if __name__ == "__main__":
    main()
