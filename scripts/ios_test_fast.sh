#!/usr/bin/env bash
set -euo pipefail

PROJECT_PATH="ios/purp tape/purp tape.xcodeproj"
SCHEME="${1:-PurpTape-Dev}"
DESTINATION="platform=iOS Simulator,id=B11FB184-AA58-4D82-A450-636672E14868,arch=arm64"

set -o pipefail
xcodebuild \
  -project "$PROJECT_PATH" \
  -scheme "$SCHEME" \
  -configuration Dev \
  -destination "$DESTINATION" \
  CODE_SIGNING_ALLOWED=NO \
  CODE_SIGNING_REQUIRED=NO \
  ONLY_ACTIVE_ARCH=YES \
  test | tee /tmp/purptape_fast_test.log
