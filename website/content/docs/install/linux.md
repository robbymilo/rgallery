---
title: Install rgallery on Linux (Debian/Ubuntu)
LinkTitle: Linux (Debian/Ubuntu)
weight: 300
logo: /logos/linux-tux.svg
---

# Install rgallery on Linux (Debian/Ubuntu)

To install rgallery on a Debian or Ubuntu Linux server, follow these steps:

Install the necessary dependencies:

```shell
sudo apt update && sudo apt install -y exiftool ffmpeg libvips
```

Navigate to the [releases page](https://github.com/robbymilo/rgallery/releases/latest).
Navigate to the latest release.
Download the latest release for your architecture:

For amd64, run:

```shell
wget https://github.com/robbymilo/rgallery/releases/download/{{< version>}}/rgallery-geo_linux-amd64
```

For arm64, run:

```shell
wget https://github.com/robbymilo/rgallery/releases/download/{{< version>}}/rgallery-geo_linux-amd64`
```

Extract the release:

```shell
tar -xzf rgallery_linux-amd64.tar.gz
```

Move the binary to a directory in your PATH, for example `/usr/local/bin`:

```shell
cp rgallery_linux-amd64 /usr/local/bin/rgallery
```

Run `rgallery` to start the server.

Navigate to the web interface at `http://<server-address>:3000`.

At this point, you should see the rgallery login page.

The default user is `admin`, and the default password is `admin`.

## Run rgallery in the background with systemd

Create a systemd service file:

```shell
sudo nano /etc/systemd/system/rgallery.service
```

Add the following to the file:

```shell
[Unit]
Description=rgallery
After=network.target

[Service]
ExecStart=/usr/local/bin/rgallery
Restart=always
User=root
WorkingDirectory=/root
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

Enable and start the service:

```shell
sudo systemctl daemon-reload
sudo systemctl enable rgallery
sudo systemctl start rgallery
```

Check the status of the service:

```shell
sudo systemctl status rgallery
```

Check the logs of the service:

```shell
sudo journalctl -u rgallery -f
```
