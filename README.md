# pipeform

## Introduction

`pipeform` is a TUI for Terraform runtime progress.

## Usage

`pipeform` as its name indicates, shall be preceded by `terraform` run, through a pipe (`|`). Only the following `terraform` commands are supported:

- `terraform refresh -json`
- `terraform plan -json`
- `terraform apply -auto-approve -json`

Note that all the commands must have the `-json` flag specified, as the tool is built on top of the [Terraform machine-readable UI](https://developer.hashicorp.com/terraform/internals/machine-readable-ui).

## FAQ

### How to exit during operation?

There are two ways to exit during operation:
- Terminate `pipeform`
- Terminate `terraform`

Though, it is highly recommended **NOT** to terminate in the middle of the run.

#### Terminate `pipeform`

There is a key bind for terminating `pipeform`. When the user hit the key to quit, `pipeform` will quit immediately, which causes the pipe to close. Since `terraform` is still running and piping out logs, it will then hit a `SIGPIPE` signal, which `terraform` has no special handling and defaults to terminate `terraform` immediately.

#### Terminate `terraform`

Terraform can be terminated by interruption (<kbd>ctrl-c</kbd>). It even has *some* graceful handling for the interruption signal.

Whilst when using `pipeform`, since the terminal is turned into *raw* mode, pressing <kbd>ctrl-c</kbd> won't send the signal at all. Instead, you'll have to send the signal manually.

Under Linux you can do something as below:

```
$ # Find out the ppid of the `pipeform`
$ ps -ef | grep pipeform
magodo     88375    8823  1 11:05 pts/7    00:00:00 pipeform
magodo     89764   49424  0 11:05 pts/6    00:00:00 grep --color pipeform
$ # 8823 is the ppid of `pipeform`
$ # Use pstree to find the pid of the preceded `terraform`
$ pstree -lpT 8823
zsh(8823)─┬─pipeform(88375)
          └─terraform(88374)───terraform-provi(88695)
$ # 88374 is the pid of `terraform`
$ # Send the signal manully
$ kill -SIGINT 88374
```

After `terraform` being interrupted in the middle, `pipeform` won't just quit. Instead, it will respond to the diagnostics sent from `terraform` (once `terraform` finishes its *graceful* handling) and display the error indicators to users.