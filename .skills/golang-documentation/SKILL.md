---
name: golang-documentation
description: Go documentation best practices — doc comments, README, CONTRIBUTING, CHANGELOG, llms.txt
---

# Go Documentation

## Overview
Write and review Go documentation following Go's own conventions and best practices.

## When to use
- When writing doc comments for Go packages, types, functions, and methods
- When creating or updating README.md, CONTRIBUTING.md, CHANGELOG.md
- When reviewing documentation for correctness and completeness

## Conventions
- **Doc comments**: Follow Go's `go doc` conventions. First sentence is a complete summary.
  ```go
  // Package math provides basic constants and mathematical functions.
  package math
  ```
- **Exported identifiers**: Always documented. Unexported: only if non-obvious.
- **README**: What, why, how to use, how to contribute.
- **CHANGELOG**: Keep a changelog following https://keepachangelog.com/
- **Code examples**: Use runnable Example functions in _test.go files.
- **llms.txt**: Provide structured context for LLMs reading the project.
