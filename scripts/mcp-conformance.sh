#!/usr/bin/env bash

set -euo pipefail

default_package="@modelcontextprotocol/conformance@0.1.16"
default_suite="active"
default_node_dir=""
if [ -n "${HOME:-}" ]; then
	default_node_dir="$HOME/.nvm/versions/node/v24.14.1/bin"
fi

usage() {
	cat <<EOF
Usage:
  MCP_CONFORMANCE_URL=http://127.0.0.1:3000/mcp $0 [--dry-run]
  $0 [--dry-run]
  $0 --help

Run the upstream MCP server conformance harness. When MCP_CONFORMANCE_URL is
not set, the script starts a small in-tree streamable HTTP fixture and stops it
after the harness exits.

Options:
  --dry-run    Print the resolved command and prerequisites, but do not run it.
  -h, --help   Show this help message.

Environment:
  MCP_CONFORMANCE_URL                Optional HTTP(S) MCP server URL.
                                     If unset, a local fixture is started.
  MCP_CONFORMANCE_SUITE              Suite to run: active, all, or pending.
                                     Default: active.
  MCP_CONFORMANCE_NODE_DIR           Node bin directory to prepend to PATH.
                                     Default: $default_node_dir when present.
  MCP_CONFORMANCE_PACKAGE            npm package spec.
                                     Default: $default_package.
  MCP_CONFORMANCE_FIXTURE_ADDR       Listen address for the local fixture.
                                     Default: 127.0.0.1:0.
  MCP_CONFORMANCE_EXPECTED_FAILURES  Optional YAML expected-failures path.
  MCP_CONFORMANCE_OUTPUT_DIR         Optional output directory for results.
  MCP_CONFORMANCE_SPEC_VERSION       Optional spec version filter.

Default harness command:
  npx -y $default_package server --url "\$MCP_CONFORMANCE_URL" --suite $default_suite
EOF
}

die() {
	echo "$0: $*" >&2
	echo >&2
	usage >&2
	exit 2
}

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
fixture_pid=
fixture_tmp=
fixture_build_cmd=()
fixture_cmd=()

cleanup() {
	if [ -n "$fixture_pid" ] && kill -0 "$fixture_pid" >/dev/null 2>&1; then
		kill "$fixture_pid" >/dev/null 2>&1 || true
		wait "$fixture_pid" >/dev/null 2>&1 || true
	fi
	if [ -n "$fixture_tmp" ]; then
		rm -rf "$fixture_tmp"
	fi
}

print_command() {
	local label=$1
	shift
	printf '%s:' "$label"
	printf ' %q' "$@"
	printf '\n'
}

dry_run=false
while [ "$#" -gt 0 ]; do
	case "$1" in
		--dry-run)
			dry_run=true
			shift
			;;
		-h | --help)
			usage
			exit 0
			;;
		*)
			die "unknown argument: $1"
			;;
	esac
done

url=${MCP_CONFORMANCE_URL:-}
suite=${MCP_CONFORMANCE_SUITE:-$default_suite}
case "$suite" in
	active | all | pending) ;;
	*) die "MCP_CONFORMANCE_SUITE must be active, all, or pending" ;;
esac

node_dir=${MCP_CONFORMANCE_NODE_DIR:-}
if [ -z "$node_dir" ] && [ -n "$default_node_dir" ] && [ -d "$default_node_dir" ]; then
	node_dir=$default_node_dir
fi
if [ -n "$node_dir" ]; then
	if [ ! -d "$node_dir" ]; then
		die "MCP_CONFORMANCE_NODE_DIR does not exist: $node_dir"
	fi
	export PATH="$node_dir:${PATH:-}"
fi

if ! command -v node >/dev/null 2>&1; then
	die "node is required; set MCP_CONFORMANCE_NODE_DIR to a Node 22+ bin directory"
fi
if ! command -v npx >/dev/null 2>&1; then
	die "npx is required; set MCP_CONFORMANCE_NODE_DIR to a Node 22+ bin directory"
fi

node_version=$(node --version)
node_major=$(node -p 'process.versions.node.split(".")[0]')
case "$node_major" in
	'' | *[!0-9]*) die "could not parse node version: $node_version" ;;
esac
if [ "$node_major" -lt 22 ]; then
	die "node $node_version is too old for fs.globSync; use Node 22+ or set MCP_CONFORMANCE_NODE_DIR=$default_node_dir"
fi

package=${MCP_CONFORMANCE_PACKAGE:-$default_package}
fixture_addr=${MCP_CONFORMANCE_FIXTURE_ADDR:-127.0.0.1:0}

if [ -z "$url" ]; then
	if [ "$dry_run" = true ]; then
		fixture_build_cmd=(go build -o "<temp>/conformance-server" ./internal/integration_testing/conformance-server)
		fixture_cmd=("<temp>/conformance-server" --addr "$fixture_addr")
		url="http://127.0.0.1:<auto>/mcp"
	else
		if ! command -v go >/dev/null 2>&1; then
			die "go is required to start the local conformance fixture"
		fi
		fixture_tmp=$(mktemp -d "${TMPDIR:-/tmp}/mcp-conformance.XXXXXX")
		trap cleanup EXIT INT TERM
		fixture_build_cmd=(go build -o "$fixture_tmp/conformance-server" ./internal/integration_testing/conformance-server)
		fixture_cmd=("$fixture_tmp/conformance-server" --addr "$fixture_addr" --url-file "$fixture_tmp/url")
		(
			cd "$repo_root"
			"${fixture_build_cmd[@]}"
		)
		(
			cd "$repo_root"
			"${fixture_cmd[@]}"
		) >"$fixture_tmp/stdout" 2>"$fixture_tmp/stderr" &
		fixture_pid=$!
		for _ in {1..100}; do
			if [ -s "$fixture_tmp/url" ]; then
				url=$(sed -n '1p' "$fixture_tmp/url")
				break
			fi
			if ! kill -0 "$fixture_pid" >/dev/null 2>&1; then
				echo "$0: local conformance fixture exited before reporting a URL" >&2
				cat "$fixture_tmp/stderr" >&2 || true
				exit 2
			fi
			sleep 0.1
		done
		if [ -z "$url" ]; then
			echo "$0: local conformance fixture did not report a URL" >&2
			cat "$fixture_tmp/stderr" >&2 || true
			exit 2
		fi
	fi
else
	case "$url" in
		http://* | https://*) ;;
		*) die "MCP_CONFORMANCE_URL must start with http:// or https://" ;;
	esac
fi

cmd=(npx -y "$package" server --url "$url" --suite "$suite")

if [ -n "${MCP_CONFORMANCE_EXPECTED_FAILURES:-}" ]; then
	if [ ! -f "$MCP_CONFORMANCE_EXPECTED_FAILURES" ]; then
		die "MCP_CONFORMANCE_EXPECTED_FAILURES does not exist: $MCP_CONFORMANCE_EXPECTED_FAILURES"
	fi
	cmd+=(--expected-failures "$MCP_CONFORMANCE_EXPECTED_FAILURES")
fi
if [ -n "${MCP_CONFORMANCE_OUTPUT_DIR:-}" ]; then
	cmd+=(--output-dir "$MCP_CONFORMANCE_OUTPUT_DIR")
fi
if [ -n "${MCP_CONFORMANCE_SPEC_VERSION:-}" ]; then
	cmd+=(--spec-version "$MCP_CONFORMANCE_SPEC_VERSION")
fi

echo "node: $node_version ($(command -v node))"
echo "npx: $(command -v npx)"
if [ "${MCP_CONFORMANCE_URL:-}" = "" ]; then
	print_command "fixture-build" "${fixture_build_cmd[@]}"
	print_command "fixture" "${fixture_cmd[@]}"
fi
print_command "command" "${cmd[@]}"

if [ "$dry_run" = true ]; then
	exit 0
fi

"${cmd[@]}"
