#!/usr/bin/env python3
"""Run tests and return structured JSON."""
import argparse, json, os, re, subprocess, sys


def parse_coverage(path):
    """Parse coverage.out into structured format: lines and percent as numbers."""
    cov_file = os.path.join(path, "coverage.out")
    if not os.path.isfile(cov_file):
        return None

    total = 0
    covered = 0
    with open(cov_file) as f:
        for line in f:
            if line.startswith("mode:"):
                continue
            parts = line.strip().split()
            if len(parts) < 3:
                continue
            try:
                num_statements = int(parts[1])
                count = int(parts[2])
                total += num_statements
                if count > 0:
                    covered += num_statements
            except (ValueError, IndexError):
                continue

    if total == 0:
        return None

    percent = round(covered / total * 100, 1)
    return {"lines": total, "percent": percent}


def parse_errors(stderr_text):
    """Parse stderr into structured error list [{file, test, message}]."""
    errors = []
    if not stderr_text:
        return errors

    for block in stderr_text.split("--- FAIL"):
        block = block.strip()
        if not block:
            continue
        lines = block.split("\n")
        # First line: "TestName (0.00s)"
        first = lines[0].strip()
        test_name = first.split()[0] if first else ""
        # Remaining lines: file:line: message
        for line in lines[1:]:
            line = line.strip()
            if not line:
                continue
            if ":" in line:
                parts = line.split(":", 2)
                errors.append({
                    "file": parts[0],
                    "test": test_name,
                    "message": ":".join(parts[1:]) if len(parts) > 2 else parts[1],
                })
    return errors


def run(args):
    path = args.path

    if os.path.isfile(os.path.join(path, "go.mod")):
        cmd = ["go", "test", "./...", "-v"]
        if args.coverage:
            cmd.append("-coverprofile=coverage.out")
    elif os.path.isfile(os.path.join(path, "pyproject.toml")):
        cmd = ["python3", "-m", "pytest", "-v", "--tb=short"]
        if args.coverage:
            cmd.append("--cov")
    else:
        print(json.dumps({"passed": 0, "failed": 0, "errors": [{"file": "", "test": "", "message": "No test framework detected"}], "coverage": None}))
        sys.exit(1)

    result = subprocess.run(cmd, capture_output=True, text=True, cwd=path)
    passed = len(re.findall(r"^(--- )?PASS|OK|\.$", result.stdout, re.MULTILINE))
    failed = len(re.findall(r"^--- FAIL|FAIL|ERROR", result.stdout, re.MULTILINE))

    coverage = None
    if args.coverage:
        coverage = parse_coverage(path)

    errors = parse_errors(result.stderr)

    return {"passed": passed, "failed": failed, "errors": errors, "coverage": coverage}


def main():
    p = argparse.ArgumentParser(description="Run tests and detect framework")
    p.add_argument("--path", default=".")
    p.add_argument("--coverage", action="store_true")
    p.add_argument("--format", default="json")
    args = p.parse_args()
    result = run(args)
    print(json.dumps(result))


if __name__ == "__main__":
    main()
