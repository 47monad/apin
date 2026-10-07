#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
temp_root=$(mktemp -d "${TMPDIR:-/tmp}/apin-consumers.XXXXXX")
trap 'rm -rf -- "$temp_root"' EXIT

modules=(
	initrs/etcdinitr
	initrs/grpcinitr
	initrs/httpinitr
	initrs/mongoinitr
	initrs/pginitr
	initrs/promgrpcinitr
	initrs/prominitr
	initrs/redisinitr
	initrs/rmqinitr
	initrs/zapinitr
)

for module_dir in "${modules[@]}"; do
	module_path="github.com/47monad/apin/$module_dir"
	consumer_dir="$temp_root/${module_dir//\//_}"
	mkdir -p "$consumer_dir"

	# promgrpcinitr has no WithConfig; use its registerer option so the generic
	# consumer still exercises the public construction API.
	constructor_option="initr.WithConfig"
	if [[ "$module_dir" == "initrs/promgrpcinitr" ]]; then
		constructor_option="initr.WithRegisterer"
	fi

	printf 'module example.invalid/consumer\n\ngo 1.27.0\n\nrequire %s v0.1.0\n\nreplace %s => %s\n' \
		"$module_path" "$module_path" "$repo_root/$module_dir" > "$consumer_dir/go.mod"
	printf 'package consumer\n\nimport initr "%s"\n\nvar _ = initr.New\nvar _ = %s\n' \
		"$module_path" "$constructor_option" > "$consumer_dir/consumer.go"
	printf 'package consumer_test\n\nimport (\n\t"testing"\n\tinitr "%s"\n)\n\nfunc TestPublicConstructionAPI(t *testing.T) {\n\t_ = initr.New\n\t_ = %s\n}\n' \
		"$module_path" "$constructor_option" > "$consumer_dir/consumer_test.go"

	echo "checking temporary consumer for $module_path"
	(
		cd "$consumer_dir"
		GOWORK=off go mod tidy
		GOWORK=off go test ./...
		GOWORK=off go list -deps -f '{{.ImportPath}}' . > deps.txt
	)

	while IFS= read -r dependency; do
		case "$dependency" in
			github.com/47monad/apin|github.com/47monad/apin/manifest|github.com/47monad/apin/initrs/*)
				if [[ "$dependency" != "$module_path" && "$dependency" != "$module_path/"* ]]; then
					echo "$module_path consumer unexpectedly imports $dependency" >&2
					exit 1
				fi
				;;
		esac
		case "$dependency" in
			cuelang.org/*|github.com/docker/*|github.com/testcontainers/*)
				echo "$module_path consumer dependency graph contains forbidden $dependency" >&2
				exit 1
				;;
		esac
		if [[ "$module_dir" == "initrs/pginitr" ]]; then
			case "$dependency" in
				go.mongodb.org/*|go.etcd.io/*|github.com/rabbitmq/*|google.golang.org/grpc*|github.com/grpc-ecosystem/*|github.com/prometheus/*)
					echo "pginitr consumer dependency graph contains forbidden $dependency" >&2
					exit 1
					;;
			esac
		fi
		if [[ "$module_dir" == "initrs/prominitr" ]]; then
			case "$dependency" in
				google.golang.org/grpc*|github.com/grpc-ecosystem/*|go.opentelemetry.io/*)
					echo "prominitr consumer dependency graph contains forbidden $dependency" >&2
					exit 1
					;;
			esac
		fi
		if [[ "$module_dir" == "initrs/redisinitr" ]]; then
			case "$dependency" in
				go.mongodb.org/*|go.etcd.io/*|github.com/rabbitmq/*|google.golang.org/grpc*|github.com/jackc/pgx/*|github.com/grpc-ecosystem/*)
					echo "redisinitr consumer dependency graph contains forbidden $dependency" >&2
					exit 1
					;;
			esac
		fi
	done < "$temp_root/${module_dir//\//_}/deps.txt"

	if [[ "$module_dir" == "initrs/pginitr" ]] && ! grep -Eq '^github\.com/jackc/pgx/v5($|/)' "$consumer_dir/deps.txt"; then
		echo "pginitr consumer dependency graph does not include pgx" >&2
		exit 1
	fi
	if [[ "$module_dir" == "initrs/promgrpcinitr" ]] && ! grep -Eq '^google\.golang\.org/grpc($|/)' "$consumer_dir/deps.txt"; then
		echo "promgrpcinitr consumer dependency graph does not include grpc" >&2
		exit 1
	fi
	if [[ "$module_dir" == "initrs/redisinitr" ]] && ! grep -Eq '^github\.com/redis/go-redis/v9($|/)' "$consumer_dir/deps.txt"; then
		echo "redisinitr consumer dependency graph does not include go-redis" >&2
		exit 1
	fi
done

consumer_dir="$temp_root/apin"
mkdir -p "$consumer_dir"
printf 'module example.invalid/apin-consumer\n\ngo 1.27.0\n\nrequire github.com/47monad/apin v0.1.0\n\nreplace github.com/47monad/apin => %s\n' \
	"$repo_root" > "$consumer_dir/go.mod"
printf 'package consumer\n\nimport (\n\t"github.com/47monad/apin"\n\t"github.com/47monad/apin/config"\n)\n\nvar _ = apin.New\nvar _ = config.Load\n' \
	> "$consumer_dir/consumer.go"
printf 'package consumer_test\n\nimport (\n\t"testing"\n\t"github.com/47monad/apin"\n\t"github.com/47monad/apin/config"\n)\n\nfunc TestLifecycleAndConfigShareRootModule(t *testing.T) {\n\t_ = apin.New\n\t_ = config.Load\n}\n' \
	> "$consumer_dir/consumer_test.go"

echo "checking temporary consumer for github.com/47monad/apin and /config"
(
	cd "$consumer_dir"
	GOWORK=off go mod tidy
	GOWORK=off go test ./...
	GOWORK=off go list -deps -f '{{.ImportPath}}' . > deps.txt
)

if ! grep -qx 'github.com/47monad/apin' "$consumer_dir/deps.txt" || ! grep -qx 'github.com/47monad/apin/config' "$consumer_dir/deps.txt"; then
	echo "root consumer did not resolve both lifecycle and config packages from apin" >&2
	exit 1
fi
