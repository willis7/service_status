# WIP: Service Status

[![CI](https://github.com/willis7/service_status/actions/workflows/ci.yml/badge.svg)](https://github.com/willis7/service_status/actions/workflows/ci.yml)

Simple Golang project to generate a static status page.

## Installation

With Make:

* `make build` - build the project on your workstation

## Usage

### `config.json`

Below is an example config which coveres the implemented checks.

``` json
{
  "services": [
    {
        "type": "ping",
        "url": "http://google.com"
    },
    {
      "type": "grep",
      "url": "https://stackoverflow.com/",
      "regex": "Ask Question"
    },
    {
      "type": "grep",
      "url": "https://www.bbc.co.uk/",
      "regex": "hello world"
    }
  ]
}
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
