#!/usr/bin/env bash
set -euo pipefail

PROJECT_PATH="ios/purp tape/purp tape.xcodeproj"
SCHEME="${1:-PurpTape-Dev}"
MODE="${2:-lint-analyze}"
CONFIGURATION="Dev"

# Modes:
# - lint-only: Run SwiftLint only
# - lint-analyze: Run SwiftLint + Static Analysis (for CI)
# - full: Run SwiftLint + Static Analysis + Tests (for local development)
SIMULATOR_ID="$(xcrun simctl list devices available | awk -F '[()]' '/iPhone 16/ {print $2; exit}')"
if [[ -n "$SIMULATOR_ID" ]]; then
  DESTINATION="platform=iOS Simulator,id=${SIMULATOR_ID}"
else
  DESTINATION="platform=iOS Simulator,name=iPhone 16"
fi

if command -v xcpretty >/dev/null 2>&1; then
  XCODEBUILD_PIPE="xcpretty"
else
  XCODEBUILD_PIPE="cat"
fi

echo "==> iOS preflight: SwiftLint"
if ! command -v swiftlint >/dev/null 2>&1; then
  echo "❌ swiftlint is not installed. Install with: brew install swiftlint"
  exit 1
fi
swiftlint lint --config .swiftlint.yml

if [[ "$MODE" == "lint-only" ]]; then
  echo "✅ iOS preflight passed (lint-only mode)"
  exit 0
fi

echo "==> iOS preflight: Static analysis"
set +e
xcodebuild \
  -project "$PROJECT_PATH" \
  -scheme "$SCHEME" \
  -configuration "$CONFIGURATION" \
  -destination "$DESTINATION" \
  analyze \
  CODE_SIGNING_ALLOWED=NO \
  CODE_SIGNING_REQUIRED=NO \
  ONLY_ACTIVE_ARCH=YES \
  COMPILER_INDEX_STORE_ENABLE=NO \
  2>&1 | $XCODEBUILD_PIPE
ANALYZE_STATUS=${PIPESTATUS[0]}
set -e

if [[ $ANALYZE_STATUS -ne 0 ]]; then
  echo "❌ Static analysis failed (exit $ANALYZE_STATUS). Collecting diagnostics..."
  set +e
  xcodebuild \
    -project "$PROJECT_PATH" \
    -scheme "$SCHEME" \
    -configuration "$CONFIGURATION" \
    -destination "$DESTINATION" \
    analyze \
    CODE_SIGNING_ALLOWED=NO \
    CODE_SIGNING_REQUIRED=NO \
    ONLY_ACTIVE_ARCH=YES \
    COMPILER_INDEX_STORE_ENABLE=NO \
    > /tmp/purptape_analyze_failure.log 2>&1
  set -e

  grep -n "error:" /tmp/purptape_analyze_failure.log | tail -n 80 || true
  echo "Full log: /tmp/purptape_analyze_failure.log"
  exit $ANALYZE_STATUS
fi

if [[ "$MODE" == "lint-analyze" ]]; then
  echo "✅ iOS preflight passed (lint-analyze mode)"
  exit 0
fi

echo "==> iOS preflight: Running tests"
xcodebuild \
  -project "$PROJECT_PATH" \
  -scheme "$SCHEME" \
  -configuration "$CONFIGURATION" \
  -destination "$DESTINATION" \
  clean test \
  CODE_SIGNING_ALLOWED=NO \
  | $XCODEBUILD_PIPE

echo "✅ iOS preflight passed (full mode with tests)"
