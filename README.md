# Sprint update

Generate markdown formatted sprint updates based on the Jira tickets were involved in the given sprint.

[![Github license](https://img.shields.io/github/license/gabor-boros/sprint-update)](https://github.com/gabor-boros/sprint-update/)

## Installation

To install `sprint-update`, use one of the [release artifacts](https://github.com/gabor-boros/sprint-update/releases) or simply run `go install https://github.com/gabor-boros/sprint-update`.

Create a new configuration file `$HOME/.sprint-update.toml` with the following content:

```toml
jira-url = "<Jira server URL>"
jira-username = "<Jira username>"
jira-password = "<Jira password>"
```

## Usage

```plaintext
Generate a sprint update in Discourse-compatible Markdown format.

Usage:
  sprint-update [flags]

Examples:
sprint-update --sprint se.253 -e

Flags:
      --config string          config file (default is $HOME/.sprint-update.yaml)
  -e, --end-of-sprint          indicate end of sprint update
  -h, --help                   help for sprint-update
      --jira-password string   jira user password
      --jira-url string        jira server URL
      --jira-username string   jira user username
  -s, --sprint string          sprint name (ex: SE.253)
      --version                show command version
```

## Development

To install everything you need for development, run the following:

```shell
$ git clone git@github.com:gabor-boros/sprint-update.git
$ cd sprint-update
$ make prerequisites
$ make deps
```

## Contributors

- [gabor-boros](https://github.com/gabor-boros)
