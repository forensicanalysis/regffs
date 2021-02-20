package regffs_test

import (
	"github.com/forensicanalysis/regffs"
	"io/fs"
	"log"
	"os"
)

func Example() {
	f, _ := os.Open("testdata/NTUSER.DAT")

	// init file system
	fsys, _ := regffs.New(f)

	fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		log.Println(path)
		return err
	})
}
