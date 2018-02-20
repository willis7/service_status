# WIP: Service Status

[![Build Status](https://travis-ci.org/willis7/service_status.svg?branch=master)](https://travis-ci.org/willis7/service_status)

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

TODO: Write more usage instructions

Client:

$ cd client && webpack-dev-server --hot --inline

build:

$ webpack

## Contributing

1. Fork it!
2. Create your feature branch: `git checkout -b my-new-feature`
3. Commit your changes: `git commit -am 'Add some feature'`
4. Push to the branch: `git push origin my-new-feature`
5. Submit a pull request :D

## TODO

* [x] load `config.json`
* [x] use template to build html
* [x] serve html
* [x] ping tests
* [x] grep tests
* [X] iterate over and test each service from config
* [X] pass results to template
* [ ] sqlite persistent data
* [ ] reactive status page

## Credits

* [CycleNerd](https://github.com/Cyclenerd/static_status) for the bash script inspiration

## License

MIT
