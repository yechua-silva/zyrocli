#!/usr/bin/env python3
"""Run tests and return structured JSON."""
import argparse, json, os, re, subprocess, sys


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
        print(json.dumps({"passed": 0, "failed": 0, "errors": "No test framework detected", "coverage": None}))
        sys.exit(1)

    result = subprocess.run(cmd, capture_output=True, text=True, cwd=path)
    passed = len(re.findall(r"^(--- )?PASS|OK|\.$", result.stdout, re.MULTILINE))
    failed = len(re.findall(r"^--- FAIL|FAIL|ERROR", result.stdout, re.MULTILINE))
    coverage = None
    if args.coverage and os.path.isfile(os.path.join(path, "coverage.out")):
        with open(os.path.join(path, "coverage.out")) as f:
            for line in f:
                m = re.match(r"^ok\s+\S+\s+[\d.]+s\s+coverage:\s+([\d.]+%)", line)
                if m:
                    coverage = m.group(1)
    return {"passed": passed, "failed": failed, "errors": result.stderr, "coverage": coverage}


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
