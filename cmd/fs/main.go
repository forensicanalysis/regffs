package main

import (
	"errors"
	"fmt"
	"github.com/forensicanalysis/fscmd"
	"github.com/forensicanalysis/regffs"
	"github.com/spf13/cobra"
	"io/fs"
	"os"
)

func main() {
	cmd := fscmd.FSCommand(func(c *cobra.Command, args []string) (fs.FS, []string, error) {
		if len(args) < 1 {
			c.Help()
			return nil, nil, errors.New("missing arguments")
		}
		f, err := os.Open(args[0])
		if err != nil {
			return nil, nil, err
		}
		fsys, err := regffs.New(f)
		return fsys, args[1:], err
	})
	cmd.Use = "regffs"
	cmd.Short = "registry viewer"
	for _, c := range cmd.Commands() {
		c.Use = c.Use + " [file]"
	}
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
