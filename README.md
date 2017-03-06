#RGoN-Exec

A simple binary that executes a command on containers via Rancher API. Useful for quick access to container contents.

##Requirements

Must have 3 environment variables present to work:

```sh
CATTLE_URL=http://000.000.0.00:8080/v1
CATTLE_ACCESS_KEY=1234567890ABCDEFGHIJ
CATTLE_SECRET_KEY=1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZabcd
```

To provide these automatically within Rancher, add the [Service Account](https://docs.rancher.com/rancher/v1.0/en/rancher-services/service-accounts/) labels to this stack:

```yml
labels:
  io.rancher.container.create_agent: 'true'
  io.rancher.container.agent.role: 'environment'
```

##Usage

Commands can be run on single containers by using the `-name` or `-id` flags:

```sh
~ ./rgon-exec -name=my-container-name -cmd="echo test"
Executing [echo test] on container [my-container-name]
test
websocket: close 1000 (normal)
~
```

Alternatively, commands can be run on a group of containers by using the `-label` flag. The same command will be executed on every matched container.

###Flags

Uses [pkg/flag](https://golang.org/pkg/flag/). Flags can be expressed as `-flag value`, `-flag=value`, or `-flag="value"`.

- `-v`: Print version. Exits immediately after.
- `-name=myContainer`: The name of a container, single string
- `-id=93jf2039ads9a`: The ID of a container, starting at the beginning (no max length)
- `-label=my-label-key`: The key of a label, like `io.rancher.stack.name`
- `-cmd=""`: Command to send. Can be formatted like `-name` but should be wrapped in quotes if the command is multiple words.
- `-debug`: **Optional** Prints debug information

##How It Works

For the following, assume that [http://localhost:8080]() is a Rancher instance, and that `1i160` is the ID of a container.

The Rancher API can be explored in a UI at [http://localhost:8080/v1](). This program makes calls to the Rancher API to find the URLs needed for making `execute` requests on containers.

The request sequence consists of 3 steps:

1. Get the `execute` URL for the given container
1. Build a tokenized request from the `execute` URL and specified command
1. Open a websocket at the tokenized endpoint, read data back

The `execute` URL can be obtained from [http://localhost:8080/v1/containers/1i160]() - it is grouped in the "actions" list. The right-side UI lets users click "execute" and will assist with building the execute command. Sending this request provides a response:

```json
{
"id": null,
"type": "hostAccess",
"links": { },
"actions": { },
"baseType": "hostAccess",
"token": "biglongtokenstringetcetc...",
"url": "ws://localhost:8080/v1/exec/"
}
```

The token in this response contains data about the target container and command. After obtaining this response, the token is sent to the websocket URL, at which time the command is executed on the container.

##Building

Run with Go: `go run executor.go -name=myContainer -cmd="echo test" -debug`

Build with Go: `go build`
