#!/usr/bin/env bash

set -euo pipefail

repo_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
workspace_dir=$(cd "$repo_dir/.." && pwd)
cache_parent="$repo_dir/.cache"
cache_dir="$cache_parent/foundation-sources"

mkdir -p "$cache_parent"
staging_dir=$(mktemp -d "$cache_parent/foundation-sources.XXXXXX")

cleanup() {
  rm -rf "$staging_dir"
}
trap cleanup EXIT

sources=(
  "ms-ts-html-validator 36feaf2da1aabcfc2b80c9c9b695c2172bc49212"
  "ms-ts-css-validator 3afd5cd9eb3ba76fca246f26b24701d1bc63bc11"
  "ms-node-validator 93d0eb75b41e8f0ca30652a184baf1cd34de579a"
  "ms-ts-browser-runtime-validator 510496919361a82acbae6ebb2fed0c7d277c3a4f"
  "ms-go-php-validator e1b9556ccd85a9028671ea0092a77f3470b78aa9"
  "ms-py-validator 1826c0aac30cfafe457eca965c5e441d0cd8e58d"
  "ms-go-code-validator b4750c0de0ad2522d6ca1bad19c1a829826e97da"
  "ms-go-linux-validator fe8a1ed76c0dd9a3db7350adeb5f1923a73c001d"
)

for source in "${sources[@]}"; do
  read -r repository commit <<<"$source"
  source_dir="$workspace_dir/$repository"
  target_dir="$staging_dir/$repository"

  if ! git -C "$source_dir" rev-parse --git-dir >/dev/null 2>&1; then
    echo "required sibling repository is missing: $source_dir" >&2
    exit 1
  fi

  if ! git -C "$source_dir" cat-file -e "$commit^{commit}" 2>/dev/null; then
    echo "required commit is missing in $repository: $commit" >&2
    echo "fetch origin/main in that repository and retry" >&2
    exit 1
  fi

  mkdir -p "$target_dir"
  git -C "$source_dir" archive "$commit" | tar -x -C "$target_dir"
  printf '%s %s\n' "$repository" "$commit" >>"$staging_dir/SOURCES.lock"
done

rm -rf "$cache_dir"
mv "$staging_dir" "$cache_dir"
trap - EXIT

echo "Prepared ${#sources[@]} pinned Foundation validator sources in $cache_dir"
