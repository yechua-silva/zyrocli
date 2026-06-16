# HelixDB — Documentación Oficial

> HelixDB Enterprise is an object-storage-backed graph database with integrated approximate vector search and BM25 full-text search. Queries are authored with Helix SDK DSLs or dynamic JSON and invoked over HTTP.

## Quickstart

```bash
curl -sSL "https://install.helix-db.com" | bash   # install the helix CLI
helix init local                                  # scaffold helix.toml + examples/
helix start dev                                   # start a local instance
helix query dev -e 'writeBatch().varAs("alice", g().addN("User", { username: "alice" })).returning(["alice"])'
helix query dev -e 'readBatch().varAs("users", g().nWithLabel("User")).returning(["users"])'
```

Full walkthrough: https://docs.helix-db.com/cli/getting-started

## Getting Started
- [Introduction](https://docs.helix-db.com/database/introduction)
- [Local Development](https://docs.helix-db.com/database/local-development)
- [Rust](https://docs.helix-db.com/database/rust-project-setup)
- [TypeScript](https://docs.helix-db.com/database/typescript-project-setup)
- [Go](https://docs.helix-db.com/database/go-project-setup)
- [Python](https://docs.helix-db.com/database/python-project-setup)
- [Roadmap](https://docs.helix-db.com/database/roadmap)
- [Release Notes](https://docs.helix-db.com/database/release-notes)

## Key Docs for Zyro
- [Querying Guide: Vector and text search](https://docs.helix-db.com/database/querying-guide/search)
- [Traversals](https://docs.helix-db.com/database/querying-guide/traversals)
- [Go SDK Setup](https://docs.helix-db.com/database/go-project-setup)
- [Data Model](https://docs.helix-db.com/database/data-model)
- [Secondary Indexes](https://docs.helix-db.com/database/indexing/secondary)
- [Vector Indexes](https://docs.helix-db.com/database/indexing/vector)
- [Text Indexes](https://docs.helix-db.com/database/indexing/text)
- [Multi-Tenancy](https://docs.helix-db.com/database/multi-tenancy)
- [Architecture](https://docs.helix-db.com/database/architecture)
