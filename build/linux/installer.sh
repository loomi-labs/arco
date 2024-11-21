#!/bin/bash
set -e

# Check for required commands
for cmd in wget unzip; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        printf "Error: Required command '%s' not found\n" "$cmd"
        exit 1
    fi
done

download_url="https://github.com/loomi-labs/arco/releases/latest/download/arco-linux.zip"

printf "You can install Arco as a systemd service. This is required for automatic backups. \n"
printf "Do you want to install Arco as a systemd service? (Y/n) "
read -r install_service

# Default to "Y" if no answer is provided
if [ -z "$install_service" ]; then
    install_service="Y"
fi

printf "Downloading Arco...\n"
if ! wget -O /tmp/arco-linux.zip "$download_url"; then
    printf "Failed to download Arco\n"
    exit 1
fi
if ! unzip -o /tmp/arco-linux.zip -d /tmp/arco-linux; then
    printf "Failed to extract archive\n"
    rm -f /tmp/arco-linux.zip
    exit 1
fi
if ! cp /tmp/arco-linux/arco ~/.local/bin/arco; then
    printf "Failed to install Arco\n"
    rm -rf /tmp/arco-linux.zip /tmp/arco-linux
    exit 1
fi

printf "\n"

if [ "$install_service" = "Y" ] || [ "$install_service" = "y" ]; then
    printf "Installing Arco as a systemd service...\n"

    # Define the service file content
    service_file="[Unit]
Description=Arco Backup Service

[Service]
Type=simple
ExecStart=%h/.local/bin/arco --hidden
Restart=on-failure
Environment=HOME=%h

[Install]
WantedBy=default.target"

    # Write the service file to the systemd directory
    echo "$service_file" | sudo tee /etc/systemd/user/arco.service > /dev/null

    # Enable and start the service
    systemctl --user enable arco
    systemctl --user start arco
fi

if command -v arco --help >/dev/null 2>&1; then
    printf "✓ Arco has been successfully installed\n"
    if [ "$install_service" = "Y" ] || [ "$install_service" = "y" ]; then
        if systemctl --user is-active --quiet arco; then
            printf "✓ Arco service is running\n"
        else
            printf "⚠ Warning: Arco service is not running\n"
        fi
    fi
    printf "\nTo get started, run: arco\n"
else
    printf "⚠ Error: Installation failed\n"
    exit 1
fi