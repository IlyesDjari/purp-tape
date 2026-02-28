# purp-tape

## iOS preflight quality gate

Run this before shipping iOS changes:

```bash
chmod +x scripts/ios_preflight.sh
./scripts/ios_preflight.sh PurpTape-Dev full
```

For fast local iteration (default: SwiftLint only):

```bash
./scripts/ios_preflight.sh PurpTape-Dev
```

For full local iOS tests without signing stalls (single simulator arch):

```bash
chmod +x scripts/ios_test_fast.sh
./scripts/ios_test_fast.sh PurpTape-Dev
```

This gate runs:
- SwiftLint (`.swiftlint.yml`)
- Xcode static analysis (`xcodebuild analyze`)
- Clean test run (`xcodebuild clean test`)