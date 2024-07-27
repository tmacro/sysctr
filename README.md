# sysctr - Simple Runner for executing containers under an init system

Sysctr is CLI client for running containers under an init system.
Container configuration is read from a JSON or YAML file.


## Usage

```shell
> ./sysctr -h
Usage: sysctr <command> [flags]

Simple container runner.

Flags:
  -h, --help                 Show context-sensitive help.
      --log-level="debug"    Set the log level.
      --log-format="text"    Set the log format. (json, text)

Commands:
  run     Run a container.
  pull    Pull a container's image.
  stop    Stop a container.
  rm      Remove a container.
```
