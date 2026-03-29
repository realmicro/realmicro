# RealMicro CLI

RealMicro CLI is the command line interface for developing [RealMicro][1] projects.

## Getting Started

[Download][2] and install **Go**. Version `1.23` or higher is required.

Installation is done by using the [`go install`][3] command.

```bash
go install github.com/realmicro/realmicro/cmd/cli/main@latest
```

## Usage

```bash
realmicro [global options] command [command options]
```

### Global Options

| Flag | Description | Default |
|------|------------|---------|
| `--reg` | Registry type (`etcd`, `mdns`) | `etcd` |
| `--reghost` | Registry address | |

### Commands

#### services

List all registered services.

```bash
realmicro --reg etcd --reghost 127.0.0.1:2379 services
```

#### describe

Describe a service and list all its endpoints.

```bash
realmicro --reg etcd --reghost 127.0.0.1:2379 describe helloworld
```

#### call

Call a service endpoint with a JSON request body.

```bash
realmicro --reg etcd --reghost 127.0.0.1:2379 call helloworld Helloworld.Call '{"name": "Asim"}'
```

[1]: https://github.com/realmicro/realmicro
[2]: https://golang.org/dl/
[3]: https://golang.org/cmd/go/#hdr-Compile_and_install_packages_and_dependencies