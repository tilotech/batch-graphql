# batch-graphql

batch-graphql is a CLI tool for easily sending batch requests to an GraphQL API
using the same query or mutation with varying variables. It is made for highly
concurrent requests to fully utilize the capacity of your and the servers
network resources.

## Installation

Either download one of the [binary releases](releases) or install from
source:

```
go install github.com/tilotech/batch-graphql
```

## Usage

Assuming there is a public GraphQL service under https://example.com/graphql
then this would be a minimal request:

```
batch-graphql -q path/to/query.graphql -i path/to/variables.jsonl -u https://example.com
```

For a more realistic example we're going to use the GraphQL API from Github. As
the API requires authorization, we're assuming that you have a
[personal access token (classic)](https://github.com/settings/tokens) available
that has at least the `read:user` scope.

First we need a file with the GraphQL query and with the request variables:

query.graphql:
```
query UserCompanyName ($loginName: String!) {
	user(login: $loginName) {
		company
	}
}
```

variables.jsonl:
```
{"loginName": "stefan-berkner-tilotech"}
{"loginName": "sami-yaseen-tilotech"}
```

The query will provide us with the company names for the specified users.

Now you can set the access token via the `--token` flag or via a configuration
file, but the preferred way is to set it using environment variables:

```
export BATCH_GRAPHQL_TOKEN=your-access-token
```

That's all that is needed to run queries against the Github GraphQL API:

```
go run main.go -q query.graphql -i variables.jsonl -u https://api.github.com/graphql
```

## Configuration

`batch-graphql` can be configured using flags, configuration files or
environment variables or any mix of those.

For a full list of supported configuration options run

```
batch-graphql help
```

By default the configuration file is located under `$HOME/.batch-graphql.json`.
Other file types (e.g. `yaml`) are also supported. You can change the location
of the configuration file by providing the `--config` flag.

Environment variables must be prefixed with `BATCH_GRAPHQL_` and then the flag
name, e.g. `BATCH_GRAPHQL_CONNECTIONS` or `BATCH_GRAPHQL_OAUTH_URL`.

### Increasing Parallel Requests

Using the `--connections` flag (`BATCH_GRAPHQL_CONNECTIONS` environment
variable), you can increase the number of parallel requests performed. By
default only lame `10` requests are send in parallel, but depending on your computer,
the network connection and the server, a value between `100` and `300` (or even
more) should also be possible.

### Authorization and Custom Headers

`batch-graphql` supports three variants of providing authorization against the
GraphQL API: (static) bearer tokens, OAuth 2.0 client credentials flow and
custom authorization using custom HTTP headers.

#### Static Bearer Token

APIs like Github support a bearer token, which can be set via the `--token` flag
(`BATCH_GRAPHQL_TOKEN` environment variable). This token will not change during
the whole batch operation.

#### OAuth 2.0 Credentials Flow

For APIs that support the OAuth 2.0 client credentials flow, you can use the
`oauth` flags:

* `--oauth.url` (`BATCH_GRAPHQL_OAUTH_URL`)
* `--oauth.clientid` (`BATCH_GRAPHQL_OAUTH_CLIENTID`)
* `--oauth.clientsecret` (`BATCH_GRAPHQL_OAUTH_CLIENTSECRET`)
* `--oauth.scope` (`BATCH_GRAPHQL_OAUTH_SCOPE`)

Or as a configuration file:

```
{
  "oauth": {
    "url": "<login url>",
    "clientid": "<client id>",
    "clientsecret": "<client secret>",
    "scope": "<scope(s)>"
  }
}
```

Using the client credentials flow, will request a bearer token from the provided
URL and use that for consecutive calls. Before the token expires, it will
automatically request a new token.

#### Custom Header

If neither of that fits, it is also possible to set custom headers:

```
batch-graphql ... --header "Authorization: value" --header "X-XYZ: foobar"
```

Or as a configuration file:

```
{
  "headers": [
    "Authorization: value",
    "X-XYZ: foobar"
  ]
}
```

### Input and Output

The input for `batch-graphql` must be a stream of JSON objects, typically a
JSON line file. You can provide the input either via the `--input` flag
(`BATCH_GRAPHQL_INPUT`) or via `stdin`.

The output will be in JSON line, where each JSON object contains the original
input, a row number, the (parsed) output as it was returned from the GraphQL
API and an `null` error. By default the output will be written to `stdout`, but
can be changed to any file using the `--output` flag (`BATCH_GRAPHQL_OUTPUT`).

If an error occurs, the error will be written either to `stderr` or to a file
provided with the `--error` flag (`BATCH_GRAPHQL_ERROR`). The format is the same
as for the regular output, except, that only the `error` message is guaranteed
to be filled and everything else is optional.