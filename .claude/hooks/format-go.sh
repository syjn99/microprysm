#!/bin/bash
# Auto-format Go files after Edit/Write tool calls.

filepath=$(cat | jq -r '.tool_input.file_path // empty')

if echo "$filepath" | grep -q '\.go$'; then
  gofmt -w "$filepath" 2>/dev/null
  goimports -w "$filepath" 2>/dev/null
fi
