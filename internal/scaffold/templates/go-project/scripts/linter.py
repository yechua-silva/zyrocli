#!/usr/bin/env python3
"""Run linter and return structured JSON."""
import argparse, json, os, subprocess


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
        return {"issues": "", "warnings": 0}

    result = subprocess.run(cmd, capture_output=True, text=True, cwd=path)
    return {"issues": result.stdout, "warnings": len(result.stderr.split("\n")) if result.stderr else 0}


def main():
    p = argparse.ArgumentParser(description="Run linter and detect config")
    p.add_argument("--path", default=".")
    p.add_argument("--fix", action="store_true")
    args = p.parse_args()
    result = run(args)
    print(json.dumps(result))


if __name__ == "__main__":
    main()
