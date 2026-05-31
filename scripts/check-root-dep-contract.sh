#!/usr/bin/env bash

# Check the root package runtime dependency contract.

set -euo pipefail

export LC_ALL=C

main_module=github.com/tmc/mcp

if [[ ! -f go.mod ]]; then
	echo "check-root-dep-contract: run from repository root" >&2
	exit 2
fi

if [[ "$(GOWORK=off go list -m)" != "$main_module" ]]; then
	echo "check-root-dep-contract: run from $main_module root" >&2
	exit 2
fi

tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

runtime_modules=$tmpdir/runtime
test_modules=$tmpdir/test
test_only_modules=$tmpdir/test-only
unexpected_modules=$tmpdir/unexpected

list_modules() {
	local args=(-deps)
	if [[ $# -gt 0 ]]; then
		args+=("$1")
	fi

	GOWORK=off go list "${args[@]}" -f '{{if and (not .Standard) .Module}}{{.Module.Path}}{{end}}' . |
		sed '/^$/d' |
		sort -u
}

list_modules >"$runtime_modules"
list_modules -test >"$test_modules"

comm -13 "$runtime_modules" "$test_modules" |
	grep -vx "$main_module" >"$test_only_modules" || true

: >"$unexpected_modules"
while IFS= read -r module; do
	if [[ "$module" == "$main_module" ]]; then
		continue
	fi
	case "$module" in
	golang.org/x/* | github.com/gorilla/websocket | github.com/santhosh-tekuri/jsonschema/v5)
		;;
	*)
		echo "$module" >>"$unexpected_modules"
		;;
	esac
done <"$runtime_modules"

if [[ -s "$unexpected_modules" ]]; then
	echo "unexpected root runtime modules:" >&2
	sed 's/^/  /' "$unexpected_modules" >&2
	echo >&2
	echo "go mod why evidence:" >&2
	while IFS= read -r module; do
		GOWORK=off go mod why -m "$module" | sed 's/^/  /' >&2
	done <"$unexpected_modules"
	exit 1
fi

echo "root runtime dependency contract satisfied"
echo "runtime modules:"
grep -vx "$main_module" "$runtime_modules" | sed 's/^/  /' || true

if [[ -s "$test_only_modules" ]]; then
	echo "test-only modules:"
	sed 's/^/  /' "$test_only_modules"
fi
