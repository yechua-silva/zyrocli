#!/usr/bin/env python3
"""Run linter and return structured JSON."""
import argparse, json, os, subprocess, sys


def run(args):
    path = args.path

    if os.path.isfile(os.path.join(path, ".golangci.yml")):
        cmd = ["golangci-lint", "run", "--out-format=json"]
        if args.fix:
            cmd.append("--fix")
    elif os.path.isfile(os.path.join(path, "pyproject.toml")):
        cmd = ["ruff", "check"]
        if args.fix:
            cmd.append("--fix")
    else:
        print(json.dumps({"issues": [], "fixed": 0, "warnings": 0}))
        sys.exit(1)

    result = subprocess.run(cmd, capture_output=True, text=True, cwd=path)
    issues = []
    for line in result.stdout.strip().split("\n"):
        parts = line.split(":")
        if len(parts) >= 3:
            issues.append({"file": parts[0], "line": parts[1], "message": ":".join(parts[2:])})
    return {"issues": issues, "fixed": 1 if args.fix else 0, "warnings": len(result.stderr.split("\n")) if result.stderr else 0}


def main():
    p = argparse.ArgumentParser(description="Run linter and detect config")
    p.add_argument("--path", default=".")
    p.add_argument("--fix", action="store_true")
    args = p.parse_args()
    result = run(args)
    print(json.dumps(result))


if __name__ == "__main__":
    main()
