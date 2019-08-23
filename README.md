# kubectl-add_config

`kubectl-add_config` is a [kubectl plugin](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/), which adds a new kubeconfig file into your local kubeconfig. This plugin is useful when the configuration for your cluster is provided as full kubeconfig file from your kubernetes cluster provider.

## Requirement

This plugin is a binary built with Golang, so you can use just a cli tool, not kubectl plugin. However, this plugin is using `kubectl` in the code and you need to install `kubectl` in your machine before using this tool.

## Development

Go 1.11 or later is required because this application using go mod for dependency management.

### Build

``` shell
$ make build
```
