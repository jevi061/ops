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
    transfer: . -> /app
  deploy:
    desc: deploy tested project to remote
    command: make deploy
    deps:
      - prepare
      - build
      - test
      - upload`
	var initCmd = &cobra.Command{
		Use:   "init",
		Short: "Create a base Opsfile to start with",
		Long:  `Create an example Opsfile in current directory if not exist`,
		Run: func(cmd *cobra.Command, args []string) {
			_, err := os.Stat(opsfile)
			if !errors.Is(err, os.ErrNotExist) {
				fmt.Fprintln(os.Stderr, "There is already an Opsfile in current directory")
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
