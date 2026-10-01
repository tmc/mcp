# Experimental tools

Commands and packages in this directory are development experiments. They may
change without notice and are not part of the stable toolkit API.

From the repository root:

```sh
go build ./exp/...
```

`exp/cmd/mcp`, `exp/mcp9p`, and `exp/cmd/mcptrace-to-otel` are separate modules covered by
`make gates` and the CI module matrix. See each command's documentation for its flags and limits.
The stable package inventory and supported workflows are in the root
[README](../README.md).
