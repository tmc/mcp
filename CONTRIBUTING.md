# Contributing

Use Go 1.25 or later. Keep packages focused and APIs small. Follow the standard
library's conventions, format Go source with `gofmt`, and include runnable
examples and regression tests for behavior changes. Generate module checksums
with Go tools. Do not commit executable build artifacts.

From the repository root:

```sh
go build ./...
go vet ./...
go test -race ./...
```

`make gates` checks formatting, build, vet, race tests, module integrity, and
root static analysis across the maintained modules. It requires `staticcheck`
on PATH. CI also checks dependency allowlists, trace benchmark thresholds,
and command smoke behavior.

Changes to the protocol should use the official SDK rather than add a
second implementation here.

For a pull request, describe the behavior change, include its regression
coverage, and identify any public API changes. Report security issues using
[the security policy](SECURITY.md).
