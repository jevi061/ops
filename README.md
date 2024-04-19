## Ops

A simple pipeline tool that allows you to run shell commands on local or remote ssh servers.


## Installation

```shell
$ go install github.com/jevi061/ops@latest
```

## Features

- run commands/scripts on local machine or remote server.
- transfer files/directories between local and remote servers.

## Usage

```shell
$ ops run TASK... [flags]
```
## Concepts

#### Opsfile
The manifest file for instructing ops to run, in which you can define target servers,tasks, and environments .
When ops starts to run, it looks for the file in the current directory. You can also set the path of Opsfile using flag -f or --opsfile.
```yaml
servers:
  example:
    host: www.example.com
    port: 22
    user: root
    password: ******
# global environments to use when ops to run tasks or pipelines
environments:
  WORKING_DIR: /app
tasks:
  prepare:
    desc: prepare build directory for building
    command: mkdir build
    local: true
  build:
    desc: build project
    command: make build
  test:
    desc: test the project
    command: make test
  upload:
    desc: upload tested project to remote
    command: src -> dst
  deploy:
    desc: deploy tested project to remote
    command: make deploy
    dependencies:
      - prepare
      - build
      - test
      - upload

```

#### Servers

Visitable Servers where tasks to run on. As ops using ssh underline, servers must have sshd run and be available to visit.

#### Tasks

Simple abstract of shell commands, which you can run on servers. Here are 3 supported task variants :
- command commands to run on remote computers
- local-command commands to run on the current local computer
- upload transfer files or directories to remote computers

Each task could have its own environments defined under the task section in Opsfile, and task-associated environments will override global environments when conflicts.



# Licence

Licensed under the [MIT License](./LICENSE).