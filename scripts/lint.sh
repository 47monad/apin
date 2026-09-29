#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo_root"
export GOWORK=off

golangci_lint_version="v2.14.0"
golangci_lint_config="$repo_root/.golangci.yml"
golangci_lint_bin_dir="${GOLANGCI_LINT_BIN_DIR:-${TMPDIR:-/tmp}/apin-golangci-lint}"

if command -v golangci-lint >/dev/null 2>&1 &&
	[[ "$(golangci-lint version --short 2>/dev/null)" == "${golangci_lint_version#v}" ]]; then
	golangci_lint="$(command -v golangci-lint)"
else
	mkdir -p "$golangci_lint_bin_dir"
	echo "installing golangci-lint ${golangci_lint_version}"
	GOBIN="$golangci_lint_bin_dir" go install \
		"github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${golangci_lint_version}"
	golangci_lint="$golangci_lint_bin_dir/golangci-lint"
fi

status=0
while IFS= read -r -d '' go_mod; do
	module_dir="${go_mod%/go.mod}"
	echo "linting Go module ${module_dir#./}"
	(
		cd "$repo_root/$module_dir"
		"$golangci_lint" run --config "$golangci_lint_config" ./...
	) || status=1
done < <(find . -name go.mod -not -path './.git/*' -print0 | sort -z)

exit "$status"
