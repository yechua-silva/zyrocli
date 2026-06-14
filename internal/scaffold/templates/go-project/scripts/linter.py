#!/usr/bin/env python3
"""Run linter and return structured JSON."""
import argparse, json, os, re, subprocess, sys


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
        return {"issues": [], "fixed": 0, "warnings": 1}

    result = subprocess.run(cmd, capture_output=True, text=True, cwd=path)
    issues = []
    fixed = 0

    if cmd[0] == "golangci-lint":
        # Parse JSON output from golangci-lint
        if result.stdout.strip():
            try:
                data = json.loads(result.stdout)
                for issue in data.get("Issues", []):
                    pos = issue.get("Pos", {})
                    issues.append({
                        "file": pos.get("Filename", ""),
                        "line": str(pos.get("Line", 0)),
                        "severity": issue.get("Severity", "error"),
                        "message": issue.get("Text", ""),
                    })
            except json.JSONDecodeError:
                pass

        if args.fix:
            # Count "fixed" mentions in stderr
            for line in result.stderr.split("\n"):
                m = re.search(r"(\d+)\s+issue", line, re.IGNORECASE)
                if m:
                    fixed = int(m.group(1))
                    break
    else:
        # Ruff output: path:line:col: code Message
        for line in result.stdout.strip().split("\n"):
            line = line.strip()
            if not line:
                continue
            parts = line.split(":", 3)
            if len(parts) >= 4:
                issues.append({
                    "file": parts[0],
                    "line": parts[1],
                    "severity": "warning",
                    "message": parts[3].strip(),
                })
            elif len(parts) >= 2:
                issues.append({
                    "file": parts[0],
                    "line": parts[1] if len(parts) > 1 else "0",
                    "severity": "warning",
                    "message": ":".join(parts[2:]),
                })

        if args.fix:
            for line in result.stderr.split("\n"):
                m = re.search(r"(\d+)\s+fixed", line, re.IGNORECASE)
                if m:
                    fixed = int(m.group(1))
                    break

    warnings = len([l for l in result.stderr.split("\n") if l.strip()]) if result.stderr else 0
    return {"issues": issues, "fixed": fixed, "warnings": warnings}


def main():
    p = argparse.ArgumentParser(description="Run linter and detect config")
    p.add_argument("--path", default=".")
    p.add_argument("--fix", action="store_true")
    args = p.parse_args()
    result = run(args)
    print(json.dumps(result))


if __name__ == "__main__":
    main()
