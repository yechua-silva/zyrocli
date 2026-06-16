# Delta for Handoff Parser

## MODIFIED Requirements

### Requirement: CLI Integration

The root Cobra command MUST register a subcommand `init` that accepts one positional argument (path or `"-"`). The `init` subcommand MUST also accept two optional bool flags: `--scaffold` and `--opencode`. `zyrocli init <path>` MUST call `Parse` then `Validate`, printing a success message on success or the error on failure. If `--scaffold` is set, the scaffold engine MUST run after validation succeeds. If `--opencode` is set (implies `--scaffold`), OpenCode MUST launch in the target directory after scaffold completes.
(Previously: only Parse+Validate, no flags; this adds `--scaffold` and `--opencode` as optional extensions)

<details>
<summary>Scenarios</summary>

- **Init with valid file**: GIVEN a valid `testdata/valid.yaml`; WHEN `zyrocli init testdata/valid.yaml` runs; THEN exit code is 0 and stdout contains "OK" or "success".
- **Init with invalid file**: GIVEN a file with missing required fields; WHEN `zyrocli init broken.yaml` runs; THEN exit code is non-zero and stderr lists validation errors.
- **Init with scaffold**: GIVEN a valid handoff; WHEN `zyrocli init testdata/valid.yaml --scaffold` runs; THEN parse+validate succeeds, scaffold creates the project directory, and exit code is 0.
- **Init with scaffold + opencode**: GIVEN `opencode` is in PATH and a valid handoff; WHEN `zyrocli init testdata/valid.yaml --scaffold --opencode` runs; THEN scaffold completes and OpenCode launches in the target directory.
- **Scaffold without opencode is error**: GIVEN only `--opencode` without `--scaffold`; WHEN the command is parsed; THEN Cobra returns a flag error as `--opencode` requires `--scaffold`.
</details>
