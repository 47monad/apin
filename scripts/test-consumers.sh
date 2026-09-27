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
	initrs/prominitr
	initrs/rmqinitr
	initrs/zapinitr
)

for module_dir in "${modules[@]}"; do
	module_path="github.com/47monad/apin/$module_dir"
	consumer_dir="$temp_root/${module_dir//\//_}"
	mkdir -p "$consumer_dir"

	printf 'module example.invalid/consumer\n\ngo 1.27.0\n\nrequire %s v0.1.0\n\nreplace %s => %s\n' \
		"$module_path" "$module_path" "$repo_root/$module_dir" > "$consumer_dir/go.mod"
	printf 'package consumer\n\nimport initr "%s"\n\nvar _ = initr.New\nvar _ = initr.WithConfig\n' \
		"$module_path" > "$consumer_dir/consumer.go"
	printf 'package consumer_test\n\nimport (\n\t"testing"\n\tinitr "%s"\n)\n\nfunc TestPublicConstructionAPI(t *testing.T) {\n\t_ = initr.New\n\t_ = initr.WithConfig\n}\n' \
		"$module_path" > "$consumer_dir/consumer_test.go"

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
	done < "$temp_root/${module_dir//\//_}/deps.txt"

	if [[ "$module_dir" == "initrs/pginitr" ]] && ! grep -Eq '^github\.com/jackc/pgx/v5($|/)' "$consumer_dir/deps.txt"; then
		echo "pginitr consumer dependency graph does not include pgx" >&2
		exit 1
	fi
done
