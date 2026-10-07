#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo_root"
export GOWORK=off

mode=release
version="${1:-}"
if [[ "$version" == "--check" ]]; then
	mode=check
elif [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([+-][0-9A-Za-z.-]+)?$ ]]; then
	echo "usage: $0 --check | vMAJOR.MINOR.PATCH" >&2
	exit 2
fi
if [[ "$mode" == "release" && "$#" -ne 1 ]]; then
	echo "usage: $0 --check | vMAJOR.MINOR.PATCH" >&2
	exit 2
fi

release_modules=(
	.
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

release_tags=()
for module_dir in "${release_modules[@]}"; do
	if [[ "$module_dir" == "." ]]; then
		release_tags+=("$version")
	else
		release_tags+=("$module_dir/$version")
	fi
done

if [[ "$mode" == "release" ]]; then
	if [[ -n "$(git status --porcelain)" ]]; then
		echo "release requires a clean worktree" >&2
		exit 1
	fi
	for tag in "${release_tags[@]}"; do
		if git show-ref --verify --quiet "refs/tags/$tag"; then
			echo "tag already exists: $tag" >&2
			exit 1
		fi
	done
fi

temp_root=$(mktemp -d "${TMPDIR:-/tmp}/apin-release-check.XXXXXX")
trap 'rm -rf -- "$temp_root"' EXIT
while IFS= read -r -d '' go_mod; do
	module_dir="${go_mod%/go.mod}"
	echo "validating Go module ${module_dir#./}"
	(
		cd "$repo_root/$module_dir"
		# Validate dependent modules against this checkout, before the root release exists.
		if grep -Eq '^[[:space:]]*github\.com/47monad/apin v' go.mod; then
			check_mod="$temp_root/${module_dir//\//_}.mod"
			check_sum="${check_mod%.mod}.sum"
			cp go.mod "$check_mod"
			if [[ -f go.sum ]]; then
				# The replacement uses this checkout rather than the published root
				# module, so its version checksums are unused in the temporary graph.
				grep -v '^github\.com/47monad/apin v' go.sum > "$check_sum" || true
			fi
			go mod edit -modfile="$check_mod" -replace="github.com/47monad/apin=$repo_root"
			go mod tidy -diff -modfile="$check_mod"
			go build -modfile="$check_mod" ./...
			go vet -modfile="$check_mod" ./...
			go test -modfile="$check_mod" ./...
		else
			go mod tidy -diff
			go build ./...
			go vet ./...
			go test ./...
		fi
	)
done < <(find . -name go.mod -not -path './.git/*' -print0 | sort -z)

bash scripts/test-consumers.sh

if [[ "$mode" == "check" ]]; then
	echo "all modules and temporary consumers passed; no tags created"
	exit 0
fi

for tag in "${release_tags[@]}"; do
	git tag "$tag"
	echo "created local tag $tag"
done
echo "validation passed and local tags created; nothing was published"
