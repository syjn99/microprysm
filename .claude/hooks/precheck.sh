#!/bin/bash
# Run relevant precheck steps based on files changed since HEAD.
# Used as a Stop hook — runs after each Claude turn.

set -euo pipefail

cd "${CLAUDE_PROJECT_DIR:-.}"

# Get changed files (staged + unstaged)
changed=$(git diff --name-only HEAD 2>/dev/null || true)
if [ -z "$changed" ]; then
  exit 0
fi

go_files=$(echo "$changed" | grep '\.go$' || true)
proto_files=$(echo "$changed" | grep '\.proto$' || true)
new_files=$(git diff --name-only --diff-filter=A HEAD 2>/dev/null || true)
new_go_files=$(echo "$new_files" | grep '\.go$' || true)

failed=0

# 1. gofmt
if [ -n "$go_files" ]; then
  bad=$(echo "$go_files" | xargs gofmt -l 2>/dev/null || true)
  if [ -n "$bad" ]; then
    echo "$go_files" | xargs gofmt -w 2>/dev/null
    echo "Fixed gofmt: $bad" >&2
  fi
fi

# 2. goimports
if [ -n "$go_files" ]; then
  bad=$(echo "$go_files" | xargs goimports -l 2>/dev/null || true)
  if [ -n "$bad" ]; then
    echo "$go_files" | xargs goimports -w 2>/dev/null
    echo "Fixed goimports: $bad" >&2
  fi
fi

# 3. Gazelle (if new Go files or BUILD files changed)
build_files=$(echo "$changed" | grep 'BUILD\.bazel$' || true)
if [ -n "$new_go_files" ] || [ -n "$build_files" ]; then
  if bazel run //:gazelle -- fix 2>/dev/null; then
    echo "Ran gazelle fix" >&2
  else
    echo "Gazelle fix failed" >&2
    failed=1
  fi
fi

# 4. Proto generation (if .proto files changed)
if [ -n "$proto_files" ]; then
  if [ -x hack/update-go-pbs.sh ]; then
    hack/update-go-pbs.sh 2>/dev/null || { echo "update-go-pbs.sh failed" >&2; failed=1; }
  fi
  if [ -x hack/update-mockgen.sh ]; then
    hack/update-mockgen.sh 2>/dev/null || { echo "update-mockgen.sh failed" >&2; failed=1; }
  fi
fi

# 5. SSZ generation (if SSZ-related files changed)
ssz_files=$(echo "$go_files" | xargs grep -l 'ssz-size\|ssz-max' 2>/dev/null || true)
if [ -n "$ssz_files" ]; then
  if [ -x hack/update-go-ssz.sh ]; then
    hack/update-go-ssz.sh 2>/dev/null || { echo "update-go-ssz.sh failed" >&2; failed=1; }
  fi
fi

exit $failed
