package regffs_test

import (
	"fmt"
	"github.com/forensicanalysis/regffs"
	"io/fs"
	"os"
)

func Example() {
	f, _ := os.Open("testdata/SYSTEM")

	// init file system
	fsys, _ := regffs.New(f)

	// print all paths
	b, _ := fs.ReadFile(fsys, "ControlSet001/Control/ComputerName/ComputerName/ComputerName")
	s, _ := regffs.DecodeRegSz(b)
	fmt.Println(s)
	// Output: WKS-WIN732BITA
}
