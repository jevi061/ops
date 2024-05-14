package transfer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Validate validates transfer syntex.
func Validate(trans string) error {
	if trans == "" {
		return errors.New("empty transfer is not allowed")
	}
	fields := strings.Fields(trans)
	if len(fields) != 3 || fields[1] != "->" {
		return errors.New("incorrect file transfer syntex, use: LOCAL_SRC -> REMOTE_DIRECTORY")
	}
	return nil
}

func ParseTransfer(trans string) (string, string, error) {
	if err := Validate(trans); err != nil {
		return "", "", err
	}
	fields := strings.Fields(trans)
	return fields[0], fields[2], nil
}

func ParseTransferWithEnvs(trans string, envs map[string]string) (string, string, error) {
	if err := Validate(trans); err != nil {
		return "", "", err
	}
	fields := strings.Fields(trans)
	absSrc, err := filepath.Abs(os.Expand(fields[0], func(s string) string { return envs[s] }))
	if err != nil {
		return "", "", fmt.Errorf("resolve upload src file path failed:%w", err)
	}
	return absSrc, fields[2], nil
}
