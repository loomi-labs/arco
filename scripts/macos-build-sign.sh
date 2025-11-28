#!/bin/bash
set -e

#############################################
# macOS Build, Sign, and Package Script
#############################################
# Fill in these variables before running:

APPLE_ID="${APPLE_ID:-}"                    # Your Apple ID email
APPLE_APP_SPECIFIC_PASSWORD="${APPLE_APP_SPECIFIC_PASSWORD:-}" # App-specific password from appleid.apple.com
APPLE_TEAM_ID="${APPLE_TEAM_ID:-}"               # Your Apple Developer Team ID

# Version to build (leave empty for dev)
VERSION="${VERSION:-dev}"

#############################################
# Auto-detected variables (usually don't need to change)
#############################################

# Certificate identities (auto-detected from keychain)
APP_CERT_IDENTITY=""
INSTALLER_CERT_IDENTITY=""

#############################################
# Helper functions
#############################################

log() {
    echo ""
    echo "===================================="
    echo "$1"
    echo "===================================="
}

check_env() {
    if [ -z "$APPLE_ID" ] || [ -z "$APPLE_APP_SPECIFIC_PASSWORD" ] || [ -z "$APPLE_TEAM_ID" ]; then
        echo "ERROR: Please fill in the required environment variables at the top of this script:"
        echo "  - APPLE_ID"
        echo "  - APPLE_APP_SPECIFIC_PASSWORD"
        echo "  - APPLE_TEAM_ID"
        exit 1
    fi
}

detect_certificates() {
    log "Detecting certificates"

    echo "Available code signing identities:"
    security find-identity -v -p codesigning

    APP_CERT_IDENTITY=$(security find-identity -v -p codesigning | grep "Developer ID Application" | head -1 | grep -o '"[^"]*"' | tr -d '"')
    INSTALLER_CERT_IDENTITY=$(security find-identity -v | grep "Developer ID Installer" | head -1 | grep -o '"[^"]*"' | tr -d '"')

    if [ -z "$APP_CERT_IDENTITY" ]; then
        echo "ERROR: No 'Developer ID Application' certificate found in keychain"
        echo "Please install your Developer ID Application certificate first."
        exit 1
    fi

    echo ""
    echo "Using App certificate: $APP_CERT_IDENTITY"

    if [ -n "$INSTALLER_CERT_IDENTITY" ]; then
        echo "Using Installer certificate: $INSTALLER_CERT_IDENTITY"
    else
        echo "WARNING: No 'Developer ID Installer' certificate found - PKG will not be signed"
    fi
}

#############################################
# Main build process
#############################################

check_env
detect_certificates

cd "$(dirname "$0")/.."
REPO_ROOT=$(pwd)
echo "Working directory: $REPO_ROOT"

# 1. Build
log "Building universal binary"
PLATFORM=darwin/universal PRODUCTION=true VERSION="$VERSION" task build

log "Creating .app bundle"
task darwin:create:app:bundle

# 2. Sign the app
log "Signing .app bundle"
codesign --sign "$APP_CERT_IDENTITY" \
    --force \
    --options runtime \
    --timestamp \
    --verbose \
    bin/arco.app

log "Verifying app signature"
codesign --verify --strict --verbose=4 bin/arco.app
echo "App signature verified"

# 3. Create and sign DMG
log "Creating DMG"
# Remove existing DMG if present
rm -f bin/Arco.dmg

# Try create-dmg first (prettier result), fall back to hdiutil if permission denied
if command -v create-dmg &> /dev/null; then
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
        "bin/arco.app" 2>&1 || true
fi

# If create-dmg failed or not available, use hdiutil directly
if [ ! -f "bin/Arco.dmg" ]; then
    echo "create-dmg failed or not available, using hdiutil instead..."
    echo "(For prettier DMG, grant Full Disk Access to Terminal in System Preferences)"

    # Create a temporary directory for DMG contents
    DMG_TEMP=$(mktemp -d)
    cp -R bin/arco.app "$DMG_TEMP/"

    # Create DMG using hdiutil
    hdiutil create -volname "Arco" \
        -srcfolder "$DMG_TEMP" \
        -ov -format UDZO \
        "bin/Arco.dmg"

    rm -rf "$DMG_TEMP"
fi

if [ ! -f "bin/Arco.dmg" ]; then
    echo "ERROR: DMG creation failed"
    exit 1
fi

log "Signing DMG"
codesign --sign "$APP_CERT_IDENTITY" --timestamp --verbose bin/Arco.dmg

log "Notarizing DMG"
xcrun notarytool submit bin/Arco.dmg \
    --apple-id "$APPLE_ID" \
    --password "$APPLE_APP_SPECIFIC_PASSWORD" \
    --team-id "$APPLE_TEAM_ID" \
    --wait

log "Stapling DMG"
xcrun stapler staple bin/Arco.dmg

# 4. Download macFUSE
log "Downloading macFUSE"
mkdir -p build/darwin/resources
if [ ! -f "build/darwin/resources/macFUSE.pkg" ]; then
    curl -L https://github.com/macfuse/macfuse/releases/download/macfuse-5.1.2/macfuse-5.1.2.dmg -o /tmp/macfuse.dmg
    hdiutil attach /tmp/macfuse.dmg -mountpoint /tmp/macfuse-mount
    cp "/tmp/macfuse-mount/Install macFUSE.pkg" build/darwin/resources/macFUSE.pkg
    hdiutil detach /tmp/macfuse-mount
    rm /tmp/macfuse.dmg
    echo "macFUSE downloaded"
else
    echo "macFUSE already downloaded, skipping"
fi

# 5. Create PKG
log "Creating PKG installer"
chmod +x build/darwin/scripts/postinstall

# Clean up any existing temp files
rm -f /tmp/arco-component.pkg
rm -f bin/Arco-Installer.pkg

pkgbuild \
    --identifier com.arcobackup.arco \
    --version "$VERSION" \
    --install-location "$HOME/Applications" \
    --scripts build/darwin/scripts \
    --component bin/arco.app \
    /tmp/arco-component.pkg

productbuild \
    --distribution build/darwin/distribution.xml \
    --resources build/darwin/resources \
    --package-path /tmp \
    bin/Arco-Installer.pkg

rm /tmp/arco-component.pkg

# 6. Sign PKG (if certificate available)
if [ -n "$INSTALLER_CERT_IDENTITY" ]; then
    log "Signing PKG"
    productsign --sign "$INSTALLER_CERT_IDENTITY" \
        bin/Arco-Installer.pkg \
        bin/Arco-Installer-signed.pkg
    mv bin/Arco-Installer-signed.pkg bin/Arco-Installer.pkg
    echo "PKG signed"
else
    echo "Skipping PKG signing (no Developer ID Installer certificate)"
fi

# 7. Notarize PKG
log "Notarizing PKG"
xcrun notarytool submit bin/Arco-Installer.pkg \
    --apple-id "$APPLE_ID" \
    --password "$APPLE_APP_SPECIFIC_PASSWORD" \
    --team-id "$APPLE_TEAM_ID" \
    --wait

log "Stapling PKG"
xcrun stapler staple bin/Arco-Installer.pkg

# 8. Final verification
log "Final verification"

echo "=== DMG Signature ==="
codesign -dv --verbose=2 bin/Arco.dmg

echo ""
echo "=== PKG Signature ==="
pkgutil --check-signature bin/Arco-Installer.pkg || echo "(unsigned)"

echo ""
echo "=== DMG Notarization ==="
stapler validate bin/Arco.dmg

echo ""
echo "=== PKG Notarization ==="
stapler validate bin/Arco-Installer.pkg

log "Build complete!"
echo ""
echo "Artifacts:"
echo "  - bin/Arco.dmg (for drag-and-drop installation)"
echo "  - bin/Arco-Installer.pkg (full installer with macFUSE + LaunchAgent)"
echo ""
ls -lh bin/Arco.dmg bin/Arco-Installer.pkg
