package transfer

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Validate validates transfer syntex.
func Validate(trans string) error {
	if trans == "" {
		return errors.New("empty payload is not allowed")
	}
	fields := strings.Fields(trans)
	if len(fields) != 3 || fields[1] != "->" {
		return errors.New("incorrect payload syntex, use: LOCAL_SRC -> REMOTE_DIRECTORY or LOCAL_DIRECTORY <- REMOTE_SRC")
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

// ParsePayloadWithEnvs parses transfer directive to get source and dest from it,and source will be
// expanded using provided envs.
func ParsePayloadWithEnvs(trans string, envs map[string]string) (string, string, error) {
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

// PipeFile pipes source of file or directory to a trigger function,
// which could used to send source data to io.Write.
func PipeFile(src string) func() (io.Reader, error) {
	piper := func() (io.Reader, error) {
		//fmt.Println("pipe file:", src)
		pr, pw := io.Pipe()

		// ensure the src actually exists before trying to tar it
		if _, err := os.Stat(src); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return nil, err
		}

		gzipw := gzip.NewWriter(pw)

		tw := tar.NewWriter(gzipw)

		go func() {
			defer pw.Close()
			defer gzipw.Close()
			defer tw.Close()
			// walk path
			err := filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {
				// return on any error
				if err != nil {
					return err
				}

				if !fi.Mode().IsRegular() {
					return nil
				}

				// create a new dir/file header
				header, err := tar.FileInfoHeader(fi, fi.Name())
				if err != nil {
					return err
				}
				pre := filepath.Dir(src)
				// update the name to correctly reflect the desired destination when untaring
				header.Name = strings.TrimPrefix(strings.Replace(file, pre, "", -1), string(os.PathSeparator))
				header.Name = strings.Replace(header.Name, string(os.PathSeparator), "/", -1)

				// write the header
				if err := tw.WriteHeader(header); err != nil {
					return err
				}

				// open files for taring
				f, err := os.Open(file)
				if err != nil {
					return err
				}

				// copy file data into tar writer
				if _, err := io.Copy(tw, f); err != nil {
					return err
				}

				// manually close here after each file operation; defering would cause each file close
				// to wait until all operations have completed.
				return f.Close()
			})
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}()

		return pr, nil
	}
	return piper

}
