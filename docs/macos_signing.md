# macOS Code Signing & Notarization Guide

This guide covers macOS code signing and notarization for Arco. The build produces two artifacts:
- **Arco.dmg** - Drag-and-drop installer
- **Arco-Installer.pkg** - Full installer with macFUSE bundled and LaunchAgent setup

## Prerequisites

### Required Tools
```bash
brew install create-dmg go-task
xcode-select --install
```

### Required Certificates

You need **two** Developer ID certificates from Apple:

| Certificate Type         | Purpose                | Required For     |
|--------------------------|------------------------|------------------|
| Developer ID Application | Signing apps and DMG   | Both DMG and PKG |
| Developer ID Installer   | Signing PKG installers | PKG only         |

**To create certificates:**
1. Go to https://developer.apple.com/account/resources/certificates/list
2. Click **+** and select the certificate type
3. Create a CSR via Keychain Access (Certificate Assistant > Request a Certificate from a Certificate Authority)
4. Upload CSR, download certificate, double-click to install

### App-Specific Password

Required for notarization:
1. Go to https://appleid.apple.com
2. Security > App-Specific Passwords > Generate
3. Save the password

### Verify Installation
```bash
# Should show both certificates
security find-identity -v -p codesigning | grep "Developer ID Application"
security find-identity -v | grep "Developer ID Installer"
```

## Local Build

Use the provided script for the full build, sign, and notarization workflow:

```bash
# Edit the script first to fill in your credentials
vim scripts/macos-build-sign.sh

# Run it
./scripts/macos-build-sign.sh
```

The script will:
1. Build universal binary
2. Create and sign .app bundle
3. Create, sign, and notarize DMG
4. Download macFUSE
5. Create, sign, and notarize PKG installer

## GitHub Actions Secrets

Configure these secrets in your repository for CI builds:

| Secret                                 | Description                                                    |
|----------------------------------------|----------------------------------------------------------------|
| `MACOS_CERTIFICATE`                    | Base64 encoded Developer ID Application .p12                   |
| `MACOS_CERTIFICATE_PASSWORD`           | Password for the .p12 file                                     |
| `MACOS_INSTALLER_CERTIFICATE`          | Base64 encoded Developer ID Installer .p12                     |
| `MACOS_INSTALLER_CERTIFICATE_PASSWORD` | Password for the installer .p12                                |
| `APPLE_ID`                             | Apple ID email                                                 |
| `APPLE_APP_SPECIFIC_PASSWORD`          | App-specific password                                          |
| `APPLE_TEAM_ID`                        | Team ID from https://developer.apple.com/account/#!/membership |

### Exporting Certificates as .p12

```bash
# 1. Open Keychain Access
# 2. Select "login" keychain > "My Certificates"
# 3. Right-click certificate > Export
# 4. Save as .p12 with password
# 5. Convert to base64:
base64 -i certificate.p12 | pbcopy
```

## Troubleshooting

### Certificate not found
```bash
# List all certificates
security find-identity -v

# Re-import if missing
# Double-click the .cer file from Apple Developer portal
```

### Notarization returns "Invalid"
```bash
# Get detailed error log
xcrun notarytool log <SUBMISSION_ID> \
  --apple-id "your@email.com" \
  --password "app-specific-password" \
  --team-id "TEAMID"
```

### productsign hangs (CI)
The keychain needs proper configuration for non-interactive access. The GitHub Actions workflow handles this via `security set-key-partition-list`.

### stapler requires Xcode
```bash
sudo xcode-select -s /Applications/Xcode.app/Contents/Developer
```

## Additional Resources

- [Apple Notarization Docs](https://developer.apple.com/documentation/security/notarizing_macos_software_before_distribution)
- [Resolving Notarization Issues](https://developer.apple.com/documentation/security/notarizing_macos_software_before_distribution/resolving_common_notarization_issues)
