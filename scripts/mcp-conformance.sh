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
  $0 --help

Run the upstream MCP server conformance harness against an already running
root v1 server endpoint. This script does not start a server fixture.

Options:
  --dry-run    Print the resolved command and prerequisites, but do not run it.
  -h, --help   Show this help message.

Environment:
  MCP_CONFORMANCE_URL                Required HTTP(S) MCP server URL.
  MCP_CONFORMANCE_SUITE              Suite to run: active, all, or pending.
                                     Default: active.
  MCP_CONFORMANCE_NODE_DIR           Node bin directory to prepend to PATH.
                                     Default: $default_node_dir when present.
  MCP_CONFORMANCE_PACKAGE            npm package spec.
                                     Default: $default_package.
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
if [ -z "$url" ]; then
	die "MCP_CONFORMANCE_URL is required"
fi
case "$url" in
	http://* | https://*) ;;
	*) die "MCP_CONFORMANCE_URL must start with http:// or https://" ;;
esac

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
printf 'command:'
printf ' %q' "${cmd[@]}"
printf '\n'

if [ "$dry_run" = true ]; then
	exit 0
fi

exec "${cmd[@]}"
