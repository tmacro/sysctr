# sysctr - Simple Runner for executing containers under an init system

Sysctr is a small CLI interface that offers a Docker Compose-like experience, specifically designed for running containers as system services.
Only basic features are currently supported, with only host networking and volume mounts available.


## But why?

I needed to run containers as system services, but was unsatisfied crafting a semi-unique systemd unit file for each container.
A smarter way would be a file containing container configurations, and a generic unit file that could be used for all containers.
Initial attempts using Docker Compose showed that while the config side of things exceeded my needs the CLI interface proved
to be unwieldy to running as a service. I kept running into issue with the process failing stay to "attached" to the container and causing excessive restarts when
running in in the foreground. When trying a different pattern using `-d` and a one-shot service issues were encountered with containers not being created/stared.


## Usage

```shell
> ./sysctr -h
Usage: sysctr <command> [flags]

Simple container runner.

Flags:
  -h, --help                 Show context-sensitive help.
      --log-level="debug"    Set the log level.
      --log-format="text"    Set the log format. (json, text)
  -c, --config=PATH          Path to sysctr configuration.

Commands:
  pull      Pull a container's image.
  run       Run a container.
  status    Get the status of a container.
  stop      Stop a container.
  rm        Remove a container.
```

**Pull an image**

```shell
> ./sysctr pull --spec spec.yaml
```

**Run a container**

```shell
> ./sysctr run --spec spec.yaml
```

**Get the status of a container**

```shell
> ./sysctr status --spec spec.yaml | jq
{
  "id": "795a76b7fcea2e4cc157dc4e037cd04a96260325a66d2f5a7758e8343fde3b09",
  "config_hash": "70f9afb025cc07ecda7a199a4be953c47af3cd7ab3c34c26fdc1088baec1b5b0",
  "status": "running"
}
```

**Stop a container**

```shell
> ./sysctr stop --spec spec.yaml
```

**Remove a container**

```shell
> ./sysctr rm --spec spec.yaml
```


## Sample Systemd Unit File

```shell
> cat /etc/systemd/system/sysctr@.service
[Unit]
Description=Sysctr container service
After=docker.service
Requires=docker.service

[Service]
TimeoutStartSec=0
ExecStartPre=/usr/local/bin/sysctr pull --spec /opt/sysctr/specs/%i.yaml
ExecStart=/usr/local/bin/sysctr run --spec /opt/sysctr/specs/%i.yaml
ExecStop=/usr/local/bin/sysctr stop --spec /opt/sysctr/specs/%i.yaml
Restart=always
RestartSec=5s

[Install]
WantedBy=multi-user.target
```
