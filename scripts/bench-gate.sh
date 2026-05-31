#!/usr/bin/env bash

set -euo pipefail

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
repo_root=$(cd "$script_dir/.." && pwd)
baseline_file=${BASELINE_FILE:-"$repo_root/testdata/benchmarks/b9-baseline.txt"}
bench_time=${BENCH_TIME:-250ms}
bench_count=${BENCH_COUNT:-10}
bench_cpu=${BENCH_CPU:-1}
current_file=$(mktemp "${TMPDIR:-/tmp}/b9-current.XXXXXX")

trap 'rm -f "$current_file"' EXIT

if [ ! -f "$baseline_file" ]; then
	echo "missing baseline file: $baseline_file" >&2
	exit 1
fi

run_benchmarks() {
	(
		cd "$repo_root"
		go test -run '^$' -bench '^BenchmarkServer_HandleRequest$/PayloadSize_1024$' -benchmem -benchtime="$bench_time" -count="$bench_count" -cpu="$bench_cpu" . >"$current_file"
		go test -run '^$' -bench '^BenchmarkTokenValidation$' -benchmem -benchtime="$bench_time" -count="$bench_count" -cpu="$bench_cpu" . >>"$current_file"
	)
}

extract_values() {
	local file=$1
	local benchmark=$2
	local metric=$3

	awk -v benchmark="$benchmark" -v metric="$metric" '
		$1 == benchmark {
			for (i = 2; i <= NF; i++) {
				if ($i == metric) {
					print $(i - 1)
					next
				}
			}
		}
	' "$file"
}

median_value() {
	local file=$1
	local benchmark=$2
	local metric=$3
	local values count index lower upper

	values=$(extract_values "$file" "$benchmark" "$metric" | LC_ALL=C sort -n)
	count=$(printf '%s\n' "$values" | sed '/^$/d' | wc -l | tr -d ' ')
	if [ "$count" -eq 0 ]; then
		echo "missing $benchmark $metric in $file" >&2
		return 1
	fi
	if [ $((count % 2)) -eq 1 ]; then
		index=$((count / 2 + 1))
		printf '%s\n' "$values" | sed -n "${index}p"
		return 0
	fi

	index=$((count / 2))
	lower=$(printf '%s\n' "$values" | sed -n "${index}p")
	upper=$(printf '%s\n' "$values" | sed -n "$((index + 1))p")
	awk -v lower="$lower" -v upper="$upper" 'BEGIN { printf "%.6f\n", (lower + upper) / 2 }'
}

best_value() {
	local file=$1
	local benchmark=$2
	local metric=$3
	local value

	value=$(extract_values "$file" "$benchmark" "$metric" | LC_ALL=C sort -n | sed -n '1p')
	if [ -z "$value" ]; then
		echo "missing $benchmark $metric in $file" >&2
		return 1
	fi
	printf '%s\n' "$value"
}

comparison_value() {
	local file=$1
	local benchmark=$2
	local metric=$3

	case "$metric" in
		ns/op) best_value "$file" "$benchmark" "$metric" ;;
		*) median_value "$file" "$benchmark" "$metric" ;;
	esac
}

tolerance_factor() {
	case "$1|$2" in
		BenchmarkServer_HandleRequest/PayloadSize_1024'|ns/op') echo 5.0 ;;
		BenchmarkServer_HandleRequest/PayloadSize_1024'|B/op') echo 1.10 ;;
		BenchmarkServer_HandleRequest/PayloadSize_1024'|allocs/op') echo 1.10 ;;
		BenchmarkTokenValidation'|ns/op') echo 5.0 ;;
		BenchmarkTokenValidation'|B/op') echo 1.0 ;;
		BenchmarkTokenValidation'|allocs/op') echo 1.0 ;;
		*)
			echo "missing tolerance for $1 $2" >&2
			return 1
			;;
	esac
}

check_metric() {
	local benchmark=$1
	local metric=$2
	local baseline current factor limit ratio

	baseline=$(comparison_value "$baseline_file" "$benchmark" "$metric")
	current=$(comparison_value "$current_file" "$benchmark" "$metric")
	factor=$(tolerance_factor "$benchmark" "$metric")
	ratio=$(awk -v baseline="$baseline" -v current="$current" 'BEGIN {
		if (baseline == 0) {
			if (current == 0) {
				print "1.00x"
			} else {
				print "inf"
			}
			exit
		}
		printf "%.2fx", current / baseline
	}')

	if awk -v baseline="$baseline" -v current="$current" -v factor="$factor" 'BEGIN {
		if (baseline == 0) {
			exit(current == 0 ? 0 : 1)
		}
		exit(current <= baseline * factor ? 0 : 1)
	}'; then
		echo "ok   $benchmark $metric baseline=$baseline current=$current ratio=$ratio limit=${factor}x"
		return 0
	fi

	echo "fail $benchmark $metric baseline=$baseline current=$current ratio=$ratio limit=${factor}x" >&2
	return 1
}

run_benchmarks

status=0
for metric in ns/op B/op allocs/op; do
	check_metric BenchmarkServer_HandleRequest/PayloadSize_1024 "$metric" || status=1
	check_metric BenchmarkTokenValidation "$metric" || status=1
done

exit "$status"
