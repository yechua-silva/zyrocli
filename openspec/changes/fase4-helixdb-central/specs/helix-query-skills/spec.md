# Helix Query Skills Specification

## Purpose

Define installation and verification of Helix query skills for ad-hoc HelixDB queries from OpenCode.

## Requirements

### Requirement: Skill Installation

A script or command MUST install the following skills into the user's skills directory: `helix-query-code`, `helix-query-skills`, `helix-query-context`. Each skill MUST be a standalone script that connects to HelixDB via HTTP REST (port 6969).

#### Scenario: Install all skills
- GIVEN HelixDB is running
- WHEN the installation command is run
- THEN three skill files exist in the skills directory

### Requirement: HelixDB Reachability Check

Each skill MUST verify HelixDB is reachable before executing its query. If HelixDB is unreachable, the skill MUST print a clear error message to stderr and exit with code 1.

#### Scenario: HelixDB not reachable
- GIVEN HelixDB is down
- WHEN any helix-query-* skill runs
- THEN stderr shows "HelixDB not reachable at localhost:6969"
- AND exit code is 1

### Requirement: JSON Output

Each skill MUST output results as JSON to stdout for machine consumption.

#### Scenario: Query returns results
- GIVEN HelixDB running with data
- WHEN a helix-query-* skill runs a query
- THEN results are printed as JSON to stdout
