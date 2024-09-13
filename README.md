## Ops

A simple pipeline tool that allows you to run shell commands on local and remote ssh servers.


## Installation

```shell
$ go install github.com/jevi061/ops@latest
```

## Features

- Run: run commands/scripts on local machine or remote server.
- File Transfer: move file/directory between local and remote servers.
- Interactive Shell: open a ssh session in terminal. 

## Usage

```shell
# init 
$ ops init

# list tasks in Opsfile
$ ops list

# run
$ ops run TASK... [flags]

# run single task
$ ops run build

# run multi tasks
$ ops run build test deploy

# open interactive shell
$ ops ssh SERVER
```
## Concepts

#### Opsfile
The manifest file for instructing ops to run, in which you can define ssh servers,tasks, and environments .
When ops starts to run, it looks for the file in the current directory. You can also set the path of Opsfile using flag -f or --opsfile.
```yaml
shell: bash
fail-fast: true
servers:
  example:
    host: www.example.com
    port: 22
    user: root
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
    transfer: src -> dst
  deploy:
    desc: deploy tested project to remote
    command: make deploy
    dependencies:
      - prepare
      - build
      - test
      - upload
```

#### shell (Optional)

Set shell program for ops to use. Here are only 2 are supported:
- sh
- bash
#### fail-fast (Optioal)

Exit immediately when meet any error.


#### servers

Accessable servers where tasks to run on. As ops using ssh underline, servers must have sshd run and be available to visit.

#### tasks

Simple abstract of shell commands, which you can run on local and remote servers. Task is minmium unit to be executed in ops. 

Each task could have its own environments defined under the task section in Opsfile, and task-associated environments will override global environments when conflicts. Example:

```yaml
tasks:
  # task name
  task-name:
    # command or script of the task
    command: echo hello
    # task description
    desc:
    # run on local or remote, type: boolean
    local: true

```



## Licence

Licensed under the [MIT License](./LICENSE).