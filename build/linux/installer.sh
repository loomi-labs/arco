#!/bin/bash

download_url="https://github.com/loomi-labs/arco/releases/latest/download/arco-linux.zip"

printf "You can install Arco as a systemd service. This is required for automatic backups. \n"
printf "Do you want to install Arco as a systemd service? (Y/n) "
read -r install_service

# Default to "Y" if no answer is provided
if [ -z "$install_service" ]; then
    install_service="Y"
fi

printf "Downloading Arco...\n"
wget -O /tmp/arco-linux.zip "$download_url"
unzip -o /tmp/arco-linux.zip -d /tmp/arco-linux
sudo cp /tmp/arco-linux/arco /usr/local/bin/

install_service="Y"

if [ "$install_service" = "Y" ] || [ "$install_service" = "y" ]; then
    printf "Installing Arco as a systemd service...\n"

    # Define the service file content
    service_file="[Unit]
Description=Arco Backup Service

[Service]
Type=simple
ExecStart=/usr/local/bin/arco --hidden
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

printf "Arco has been installed.\n"