#!/usr/bin/env bash

set -euo pipefail

repo_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
compose=(
  docker compose
  --project-directory "$repo_dir"
  -f "$repo_dir/docker-compose.yml"
  -f "$repo_dir/docker-compose.foundation.yml"
)

services=(
  ms-go-validation-orchestrator
  foundation-html-validator
  foundation-node-validator
  foundation-browser-validator
  foundation-php-validator
  foundation-python-validator
  foundation-code-validator
  foundation-linux-validator
)

required_engines=(
  browser.runtime
  go.core
  html.dom
  java.compile
  java.runtime
  js.ast
  kotlin.compile
  kotlin.runtime
  linux.cli
  linux.runtime
  php.core
  python.core
  ts.ast
  ts.runtime
)

for service in "${services[@]}"; do
  container_id=$("${compose[@]}" ps -q "$service")
  if [[ -z "$container_id" ]]; then
    echo "service is not running: $service" >&2
    exit 1
  fi

  health=$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}missing{{end}}' "$container_id")
  if [[ "$health" != "healthy" ]]; then
    echo "service is not healthy: $service ($health)" >&2
    exit 1
  fi
done

engines=$("${compose[@]}" exec -T ms-go-validation-orchestrator \
  wget -q -O - http://127.0.0.1:8080/api/v1/engines)

for engine in "${required_engines[@]}"; do
  if ! grep -Fq "\"$engine\"" <<<"$engines"; then
    echo "required engine is not configured: $engine" >&2
    exit 1
  fi
done

echo "Foundation stack is healthy: ${#services[@]} services, ${#required_engines[@]} required engines."
