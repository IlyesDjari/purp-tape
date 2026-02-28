#!/usr/bin/env bash
set -euo pipefail

PROJECT_PATH="ios/purp tape/purp tape.xcodeproj"
SCHEME="${1:-PurpTape-Dev}"
MODE="${2:-lint-only}"
CONFIGURATION="Dev"
DESTINATION="platform=iOS Simulator,name=iPhone 16"

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
  echo "✅ iOS preflight lint-only gate passed"
  exit 0
fi

echo "==> iOS preflight: Static analysis"
xcodebuild \
  -project "$PROJECT_PATH" \
  -scheme "$SCHEME" \
  -configuration "$CONFIGURATION" \
  -destination "$DESTINATION" \
  analyze \
  CODE_SIGNING_ALLOWED=NO \
  | $XCODEBUILD_PIPE

echo "==> iOS preflight: Clean test run"
xcodebuild \
  -project "$PROJECT_PATH" \
  -scheme "$SCHEME" \
  -configuration "$CONFIGURATION" \
  -destination "$DESTINATION" \
  clean test \
  CODE_SIGNING_ALLOWED=NO \
  | $XCODEBUILD_PIPE

echo "✅ iOS preflight quality gate passed"
