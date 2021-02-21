<h1 align="center">regffs</h1>

<p  align="center">
  <a href="https://godocs.io/github.com/forensicanalysis/regffs"><img src="https://godocs.io/github.com/forensicanalysis/regffs?status.svg" alt="doc" /></a>
</p>

Read Windows registry files (regf) as [io/fs.FS](https://golang.org/pkg/io/fs/#FS).

# Example

``` go
func main() {
	f, _ := os.Open("testdata/SYSTEM")

	// init file system
	fsys, _ := regffs.New(f)

	// print all paths
	b, _ := fs.ReadFile(fsys, "ControlSet001/Control/ComputerName/ComputerName/ComputerName")
	s, _ := regffs.DecodeRegSz(b)
	fmt.Println(s)
	// Output: WKS-WIN732BITA
}
```

# License

`testdata` is from https://github.com/log2timeline/plaso and therefore licenced
under [Apache License 2.0](testdata/LICENSE). The remaining files are [MIT](LICENSE) licensed.
