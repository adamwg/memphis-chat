# memphis-chat - An example application for Memphis.dev

This is a simple chat app using the Memphis message broker to distribute messages.

## Building

```console
$ go build .
```

## Running

First, create a station called `memphis-chat` on your Memphis server. Then, run the following:

```console
$ export MEMPHIS_ADDR=<your memphis server>
$ export MEMPHIS_USER=<your memphis username>
$ export MEMPHIS_PASSWORD=<your memphis password>
$ ./memphis-chat -username=<chat username>
```

## Quitting

Press Ctrl+C to exit.
