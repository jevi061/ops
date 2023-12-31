## Ops

A simple pipeline tool that allows you to:
- run shell commands on local or remote ssh servers
- transfer files or directories to remotes

## Installation

```shell
$ go install github.com/jevi061/ops@latest
```
## Usage

```shell
$ ops run [task|pipeline...] [flags]
```
## Concepts

#### Opsfile
The manifest file for instructing ops to run, in which you can define target servers,tasks, and pipelines .
When ops starts to run, it looks for the file in the current directory. You can also set the path of Opsfile using flag -f or --opsfile.
```yaml
servers:
  - www.example.com
    port: 22
    user: root
    password: ******
# global environments to use when ops to run tasks or pipelines
environments:
  WORKING_DIR: /app
tasks:
  prepare:
    desc: prepare build directory for building
    local-cmd: mkdir build
  build:
    desc: build project
    cmd: make build
  test:
    desc: test the project
    cmd: make test
  upload:
    desc: upload tested project to remote
    upload:
      src: .
      dest: /app
  deploy:
    desc: deploy tested project to remote
    cmd: make deploy
pipelines:
  deploy-project:
    - build
    - test
    - upload
    - deploy
```

#### Servers

Visitable Servers where tasks or pipelines to run on. As ops using ssh underline, servers must have sshd run and be available to visit.

#### Tasks

Simple abstract of shell commands, which you can run on servers. Here are 3 supported task variants :
- cmd commands to run on remote computers
- local-cmd commands to run on the current local computer
- upload transfer files or directories to remote computers

Each task could have its own environments defined under the task section in Opsfile, and task-associated environments will override global environments when conflicts

#### Pipelines

Series of tasks to run.

# Licence

Licensed under the [MIT License](./LICENSE).