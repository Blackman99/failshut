# Contributing

Keep changes inside [GOAL.md](./GOAL.md). New rules need a fixture under `testdata/<rule-id>/` and a test in `rules/`.

```bash
gofmt -w .
go vet ./...
go test -race ./...
```

Commit as yourself. Releases are tagged `vX.Y.Z` (GoReleaser). Design language is locked in `site/DESIGN.md` — do not introduce SaaS dark+cyan templates or registered hex/fonts from other factory projects.
