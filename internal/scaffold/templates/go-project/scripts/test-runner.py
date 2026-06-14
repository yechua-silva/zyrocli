#!/usr/bin/env python3
"""Run tests and return structured JSON."""
import argparse, json, os, re, subprocess


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
        return {"passed": 0, "failed": 0, "output": "", "errors": "No test framework detected"}

    result = subprocess.run(cmd, capture_output=True, text=True, cwd=path)
    passed = len(re.findall(r"^(--- )?PASS|OK|\.$", result.stdout, re.MULTILINE))
    failed = len(re.findall(r"^--- FAIL|FAIL|ERROR", result.stdout, re.MULTILINE))
    return {"passed": passed, "failed": failed, "output": result.stdout, "errors": result.stderr}


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
