#!/usr/bin/env bash
set -euo pipefail

token="${INTERNAL_API_TOKEN:-}"
if [[ -z "$token" || "$token" =~ [[:space:]] ]]; then
	echo "native run requires INTERNAL_API_TOKEN without whitespace" >&2
	exit 2
fi

token_lower="$(printf '%s' "$token" | tr '[:upper:]' '[:lower:]')"
case "$token_lower" in
	change-me|change_me|changeme|placeholder|your-token|your_token|replace-me|replace_me|example|example-token|secret|token|password|default|none|null)
		echo "native run rejects placeholder INTERNAL_API_TOKEN values" >&2
		exit 2
		;;
esac

export HOST=127.0.0.1
export PORT=18102

exec go run ./cmd/ms-go-validation-orchestrator
