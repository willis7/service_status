# Service Status

[![CI](https://github.com/willis7/service_status/actions/workflows/ci.yml/badge.svg)](https://github.com/willis7/service_status/actions/workflows/ci.yml)

Simple Golang project to generate a static status page.

## Installation

```bash
task build
```

## Usage

### Commands

- `task run` - run the application in development mode
- `task build` - build the binary to `bin/status`
- `task start` - run the compiled binary
- `task test` - run tests
- `task check` - run vet, lint, and tests
- `task --list` - list all available tasks

### CLI

Run `service_status --help` for full CLI options:

- `serve` - start the HTTP status server (default port 8080)
- `check` - run a single status check and print results (useful for CI/CD)

### Configuration

Configuration can be provided as YAML or JSON. Create a `config.yaml` file:

```yaml
services:
  - type: ping
    url: http://google.com
    name: Google

  - type: grep
    url: https://stackoverflow.com/
    regex: "Ask Question"
    name: Stack Overflow

  - type: grep
    url: https://www.bbc.co.uk/
    regex: "hello world"
    name: BBC News

# Optional settings
alert_cooldown: 300
storage_path: "status.db"
maintenance_message: "System under maintenance"
incident_history_limit: 10
min_incident_duration: 60

# Notifiers (optional)
notifiers:
  - type: webhook
    webhook_url: http://your-webhook-endpoint.com/notify
```

#### Service Types

- `ping` - HTTP GET request, returns UP if response received
- `grep` - HTTP GET + regex match on body, returns UP if pattern found

#### Environment Variables

Any config option can be overridden with environment variables using the `SERVICE_STATUS_` prefix:

```bash
SERVICE_STATUS_PORT=9000 service_status serve
SERVICE_STATUS_STORAGE_PATH=/data/status.db service_status serve
```

## Contributing

1. Fork it!
2. Create your feature branch: `git checkout -b my-new-feature`
3. Commit your changes: `git commit -am 'Add some feature'`
4. Push to the branch: `git push origin my-new-feature`
5. Submit a pull request :D

## Credits

* [CycleNerd](https://github.com/Cyclenerd/static_status) for the bash script inspiration

## License

MIT
