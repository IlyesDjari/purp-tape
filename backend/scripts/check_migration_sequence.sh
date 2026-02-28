#!/usr/bin/env bash
set -euo pipefail

MIGRATIONS_DIR="${MIGRATIONS_DIR:-migrations}"

if [[ ! -d "$MIGRATIONS_DIR" ]]; then
  echo "Migrations directory not found: ${MIGRATIONS_DIR}" >&2
  exit 1
fi

errors=0
tmp_versions="$(mktemp)"
trap 'rm -f "$tmp_versions"' EXIT

while IFS= read -r migration; do
  file_name="$(basename "$migration")"

  if [[ ! "$file_name" =~ ^([0-9]{3})_ ]]; then
    echo "Invalid migration filename (expected NNN_description.sql): ${file_name}" >&2
    errors=1
    continue
  fi

  echo "${BASH_REMATCH[1]} ${file_name}" >> "$tmp_versions"
done < <(find "$MIGRATIONS_DIR" -maxdepth 1 -type f -name "*.sql" | sort)

duplicate_versions="$(cut -d' ' -f1 "$tmp_versions" | sort | uniq -d)"
if [[ -n "$duplicate_versions" ]]; then
  while IFS= read -r version; do
    [[ -z "$version" ]] && continue
    conflict_files="$(grep "^${version} " "$tmp_versions" | cut -d' ' -f2- | paste -sd ', ' -)"
    echo "Duplicate migration version ${version}: ${conflict_files}" >&2
  done <<< "$duplicate_versions"
  errors=1
fi

if [[ "$errors" -ne 0 ]]; then
  exit 1
fi

echo "Migration naming/sequence check passed"
