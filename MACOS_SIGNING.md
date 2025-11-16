# macOS Code Signing & Notarization - Local Testing Guide

This guide provides comprehensive instructions for testing macOS code signing and notarization locally on your Mac.

## Prerequisites Installation

### 1. Install Homebrew (if not already installed)
```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

### 2. Install Required Tools
```bash
# Install create-dmg for creating DMG installers
brew install create-dmg

# Install Task (if not already installed)
brew install go-task

# Xcode Command Line Tools (includes codesign, notarytool)
xcode-select --install
```

## Certificate Setup

### 1. Get Your Developer ID Certificate

**Option A: If you have Xcode installed**
1. Open Xcode
2. Go to **Preferences** → **Accounts**
3. Select your Apple ID
4. Click **Manage Certificates**
5. Click the **+** button and select **Developer ID Application**
6. The certificate will be downloaded to your keychain

**Option B: From Apple Developer Portal**
1. Go to https://developer.apple.com/account/resources/certificates/list
2. Click **+** to create a new certificate
3. Select **Developer ID Application**
4. Follow the instructions to create a Certificate Signing Request (CSR):
   ```bash
   # This will open Keychain Access
   open /Applications/Utilities/Keychain\ Access.app

   # In Keychain Access:
   # - Menu: Keychain Access → Certificate Assistant → Request a Certificate from a Certificate Authority
   # - User Email: your@email.com
   # - Common Name: Your Name
   # - Request is: Saved to disk
   ```
5. Upload the CSR to the Developer Portal
6. Download the certificate and double-click to install it in your keychain

### 2. Get App-Specific Password for Notarization

1. Go to https://appleid.apple.com
2. Sign in with your Apple ID
3. Under **Security** → **App-Specific Passwords**
4. Click **Generate an app-specific password**
5. Label it (e.g., "Arco Notarization")
6. **Save this password** - you'll need it for notarization

### 3. Verify Certificate Installation

```bash
# List all code signing certificates
security find-identity -v -p codesigning

# You should see output like:
# 1) ABC123... "Developer ID Application: Your Name (TEAMID123)"
```

## Building and Signing the App

### 1. Build the Application

```bash
cd /path/to/arco

# Build universal binary
PLATFORM=darwin/universal VERSION=test-local PRODUCTION=true task build

# Create .app bundle
task darwin:create:app:bundle
```

### 2. Sign the .app Bundle

```bash
# Extract your certificate identity
CERT_IDENTITY=$(security find-identity -v -p codesigning | grep "Developer ID Application" | head -1 | grep -o '"[^"]*"' | tr -d '"')

echo "Found certificate: $CERT_IDENTITY"

# Sign the app
codesign --sign "$CERT_IDENTITY" \
  --force \
  --options runtime \
  --timestamp \
  --verbose \
  bin/arco.app

# Verify the signature
codesign --verify --deep --strict --verbose=4 bin/arco.app

# Check signature details
codesign -dv --verbose=4 bin/arco.app
```

### 3. Create and Sign DMG

```bash
# Create DMG
create-dmg \
  --volname "Arco" \
  --volicon "build/darwin/icons.icns" \
  --window-pos 200 120 \
  --window-size 600 400 \
  --icon-size 100 \
  --icon "arco.app" 175 190 \
  --hide-extension "arco.app" \
  --app-drop-link 425 190 \
  --no-internet-enable \
  "bin/Arco.dmg" \
  "bin/arco.app"

# Sign the DMG
codesign --sign "$CERT_IDENTITY" \
  --timestamp \
  --verbose \
  bin/Arco.dmg

# Verify DMG signature
codesign -dv --verbose=4 bin/Arco.dmg
```

## Notarization Process

### 1. Store Credentials (One-time Setup)

```bash
# Store your credentials in the keychain
xcrun notarytool store-credentials "arco-notary-profile" \
  --apple-id "your-apple-id@example.com" \
  --team-id "YOUR_TEAM_ID" \
  --password "your-app-specific-password"

# Find your Team ID at: https://developer.apple.com/account/#!/membership
```

### 2. Submit for Notarization

```bash
# Submit DMG for notarization
xcrun notarytool submit bin/Arco.dmg \
  --keychain-profile "arco-notary-profile" \
  --wait

# This will output something like:
# Submission ID received
#   id: abc123-def456-ghi789
# Processing complete
#   status: Accepted (or Invalid)
```

### 3. Check Notarization Status

```bash
# Check history
xcrun notarytool history --keychain-profile "arco-notary-profile"

# Get detailed log for a specific submission
xcrun notarytool log abc123-def456-ghi789 \
  --keychain-profile "arco-notary-profile" \
  notarization_log.json

# View the log (requires jq)
cat notarization_log.json | jq .

# Or view raw JSON
cat notarization_log.json
```

### 4. Staple the Ticket (if notarization succeeded)

```bash
# Staple the notarization ticket to the DMG
xcrun stapler staple bin/Arco.dmg

# Verify stapling
xcrun stapler validate bin/Arco.dmg
```

## Troubleshooting Commands

### Check What's Wrong with Signing

```bash
# Detailed signature verification
codesign --verify --deep --strict --verbose=4 bin/arco.app

# Check if hardened runtime is enabled
codesign -dv bin/arco.app 2>&1 | grep -i runtime

# List all signatures in the app
codesign -dvvv bin/arco.app

# Test Gatekeeper assessment (how macOS will treat the app)
spctl --assess --type execute --verbose bin/arco.app
```

### Check Notarization Issues

```bash
# After getting "Invalid" status, fetch detailed errors:
SUBMISSION_ID="your-submission-id-here"

xcrun notarytool log "$SUBMISSION_ID" \
  --keychain-profile "arco-notary-profile" \
  error_log.json

# Pretty print the errors (requires jq)
cat error_log.json | jq '.issues'

# Or view the raw issues
cat error_log.json
```

### Verify DMG Contents

```bash
# Mount the DMG and check the app inside
hdiutil attach bin/Arco.dmg -readonly -mountpoint /tmp/arco_check

# Verify the app inside the DMG
codesign --verify --deep --strict --verbose=4 /tmp/arco_check/arco.app

# Unmount
hdiutil detach /tmp/arco_check
```

## Complete Test Script

Here's a complete script you can run to test everything. Save this as `test-signing.sh`:

```bash
#!/bin/bash
set -e

echo "=== 1. Building Application ==="
PLATFORM=darwin/universal VERSION=test-local PRODUCTION=true task build
task darwin:create:app:bundle

echo ""
echo "=== 2. Checking Available Certificates ==="
security find-identity -v -p codesigning

echo ""
echo "=== 3. Extracting Certificate Identity ==="
CERT_IDENTITY=$(security find-identity -v -p codesigning | grep "Developer ID Application" | head -1 | grep -o '"[^"]*"' | tr -d '"')
echo "Using: $CERT_IDENTITY"

echo ""
echo "=== 4. Signing .app Bundle ==="
codesign --sign "$CERT_IDENTITY" \
  --force \
  --options runtime \
  --timestamp \
  --verbose \
  bin/arco.app

echo ""
echo "=== 5. Verifying .app Signature ==="
codesign --verify --deep --strict --verbose=4 bin/arco.app
codesign -dv --verbose=4 bin/arco.app

echo ""
echo "=== 6. Creating DMG ==="
create-dmg \
  --volname "Arco" \
  --volicon "build/darwin/icons.icns" \
  --window-pos 200 120 \
  --window-size 600 400 \
  --icon-size 100 \
  --icon "arco.app" 175 190 \
  --hide-extension "arco.app" \
  --app-drop-link 425 190 \
  --no-internet-enable \
  "bin/Arco.dmg" \
  "bin/arco.app" || echo "DMG creation finished"

echo ""
echo "=== 7. Signing DMG ==="
codesign --sign "$CERT_IDENTITY" \
  --timestamp \
  --verbose \
  bin/Arco.dmg

echo ""
echo "=== 8. Verifying DMG Signature ==="
codesign -dv --verbose=4 bin/Arco.dmg

echo ""
echo "=== 9. Submitting for Notarization ==="
xcrun notarytool submit bin/Arco.dmg \
  --keychain-profile "arco-notary-profile" \
  --wait

echo ""
echo "=== 10. Stapling Ticket ==="
xcrun stapler staple bin/Arco.dmg

echo ""
echo "=== 11. Final Verification ==="
xcrun stapler validate bin/Arco.dmg
spctl --assess --type execute --verbose bin/Arco.dmg

echo ""
echo "✅ All steps completed!"
```

**To use this script:**

1. Save it as `test-signing.sh` in the project root
2. Make it executable: `chmod +x test-signing.sh`
3. Run it: `./test-signing.sh`

## GitHub Actions Secrets Reference

For the GitHub Actions workflow, you'll need these secrets configured in your repository:

| Secret Name | Description | How to Get It |
|-------------|-------------|---------------|
| `MACOS_CERTIFICATE` | Base64 encoded .p12 certificate file | Export from Keychain Access, then `base64 -i certificate.p12 | pbcopy` |
| `MACOS_CERTIFICATE_PASSWORD` | Password for the .p12 file | Password you set when exporting the certificate |
| `APPLE_ID` | Your Apple ID email | The email associated with your Apple Developer account |
| `APPLE_APP_SPECIFIC_PASSWORD` | App-specific password | Generated at https://appleid.apple.com |
| `APPLE_TEAM_ID` | Your Apple Team ID | Found at https://developer.apple.com/account/#!/membership |

### Exporting Certificate as .p12

1. Open **Keychain Access** app
2. Select **login** keychain (left sidebar)
3. Select **My Certificates** category (left sidebar)
4. Find your "Developer ID Application" certificate
5. Right-click → **Export "Developer ID Application..."**
6. Save as `.p12` file with a password
7. Convert to base64 for GitHub secret:
   ```bash
   base64 -i /path/to/certificate.p12 | pbcopy
   ```
8. Paste into GitHub → Settings → Secrets → New repository secret

## Common Issues and Solutions

### Issue: "Developer ID Application" certificate not found

**Solution:**
```bash
# List all your certificates
security find-identity -v

# Import the certificate if it's missing
# Double-click your downloaded certificate file
```

### Issue: Notarization returns "Invalid"

**Solution:**
```bash
# Get the submission ID from the notarization output
SUBMISSION_ID="your-id-here"

# Fetch detailed error log
xcrun notarytool log "$SUBMISSION_ID" \
  --keychain-profile "arco-notary-profile" \
  error_log.json

# Check the errors
cat error_log.json
```

### Issue: "binary is not signed" error during notarization

**Solution:** Make sure you're signing with the exact certificate identity:
```bash
# Get the exact identity name
security find-identity -v -p codesigning

# Use the full identity string (not just "Developer ID Application")
CERT_IDENTITY="Developer ID Application: Your Company Name (TEAMID123)"
codesign --sign "$CERT_IDENTITY" ...
```

### Issue: Hardened runtime not enabled

**Solution:** Always use `--options runtime` flag:
```bash
codesign --sign "$CERT_IDENTITY" \
  --options runtime \
  --timestamp \
  bin/arco.app
```

## Additional Resources

- [Apple Notarization Documentation](https://developer.apple.com/documentation/security/notarizing_macos_software_before_distribution)
- [Resolving Common Notarization Issues](https://developer.apple.com/documentation/security/notarizing_macos_software_before_distribution/resolving_common_notarization_issues)
- [Code Signing Guide](https://developer.apple.com/library/archive/documentation/Security/Conceptual/CodeSigningGuide/Introduction/Introduction.html)
- [notarytool Documentation](https://developer.apple.com/documentation/security/notarizing_macos_software_before_distribution/customizing_the_notarization_workflow)
