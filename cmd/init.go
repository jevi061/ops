package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func NewInitCmd() *cobra.Command {
	var opsfile = "./Opsfile.yml"
	const base = `
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
    local-cmd: mkdir build
  build:
    desc: build project
    cmd: make build
  test:
    desc: test the project
    cmd: make test
  upload:
    desc: upload tested project to remote
    cmd: . -> /app
  deploy:
    desc: deploy tested project to remote
    cmd: make deploy
    deps:
      - prepare
      - build
      - test
      - upload

	`
	var initCmd = &cobra.Command{
		Use:   "init",
		Short: "Init a base Opsfile",
		Long:  `Create an example Opsfile`,
		Run: func(cmd *cobra.Command, args []string) {
			_, err := os.Stat(opsfile)
			if !errors.Is(err, os.ErrNotExist) {
				fmt.Fprintln(os.Stderr, "There is already a Opsfile in current directory")
				os.Exit(1)
			}
			f, err := os.Create(opsfile)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			defer f.Close()
			if _, err := f.WriteString(base); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		},
	}
	return initCmd
}
