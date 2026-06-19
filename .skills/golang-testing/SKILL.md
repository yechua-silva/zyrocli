---
name: golang-testing
description: Go testing best practices — table-driven tests, subtests, mocking, golden files
---

# Go Testing

## Overview
Write idiomatic Go tests following best practices and conventions.

## When to use
- When writing unit tests for Go packages
- When writing integration tests
- When writing table-driven tests
- When using test coverage to identify untested paths

## Conventions
- **Table-driven tests**: Use anonymous structs for test cases.
  ```go
  func TestFoo(t *testing.T) {
      tests := []struct {
          name string
          input string
          want  string
      }{
          {"empty", "", ""},
          {"hello", "hello", "HELLO"},
      }
      for _, tt := range tests {
          t.Run(tt.name, func(t *testing.T) {
              got := Foo(tt.input)
              if got != tt.want {
                  t.Errorf("Foo(%q) = %q, want %q", tt.input, got, tt.want)
              }
          })
      }
  }
  ```
- **Test helpers**: Use `t.Helper()` for helper functions
- **Subtests**: Use `t.Run()` for grouped test cases
- **Coverage**: Aim for meaningful coverage, not 100% for its own sake
- **Golden files**: Use `testdata/golden/` for expected output
- **Mocking**: Use interfaces, not concrete types, for testability
