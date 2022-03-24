# My own Dapr

A simple example showing how you can create your own custom Dapr sidecar binary (`daprd`).

## How it works

Go leverages Git repositories as dependencies. This means that it is possible to use Dapr as a library
in other Go programs. To make our own customer Dapr sidecar, all we need to do is create a small program
with just the components we want included and starts the runtime. Easy, right!

## Changine the Dapr runtime version or included components

This project already has a customized `daprd` program that only includes Redis as an example.
To customize which components are included in the binary, simply copy this
[`main.go`](https://github.com/dapr/dapr/blob/master/cmd/daprd/main.go) into `cmd/daprd` and add/remove
component registrations. You can even provide your own components.

Next, overwrite the entire contents `go.mod` with this snippet
(retrofit the module name and desired Dapr version).

```
module github.com/pkedy/my_own_dapr

go 1.17

require (
	github.com/dapr/components-contrib v1.6.0
	github.com/dapr/dapr v1.6.0
)
```

Then from the terminal, run `go mod tidy --compat=1.17` and the rest of the required dependencies will be included.

Finally, make sure that `DAPR_RELEASE` in the `Makefile` is set to the Dapr version you want (e.g., `1.6.0`).
This is so that the docker image tag includes the correct version.

## Building

You'll want to tweak the `Makefile` to change the docker image tag to your liking. Then run `make docker`.
