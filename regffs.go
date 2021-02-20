package regffs

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/xlab/treeprint"
)

func main() {
	b, _ := os.ReadFile("testdata/NTUSER.DAT")
	f := bytes.NewReader(b)

	regffs := New(f)

	tree := treeprint.New()

	err := fs.WalkDir(regffs, ".", func(path string, d fs.DirEntry, err error) error {
		info, _ := d.Info()
		fmt.Println(">", path, d.IsDir(), info.Size())
		return err
	})
	if err != nil {
		log.Fatal(err)
	}
	return
	infos, _ := fs.ReadDir(regffs, ".")
	for _, info := range infos {
		fmt.Println(info.Name())
	}
	return

	// handleCell(int64(header.RootKeyOffset()+0x1000), f, regf, tree)

	os.Remove("tree")

	out := bytes.ReplaceAll(tree.Bytes(), []byte(" "), []byte(""))

	os.WriteFile("tree", out, fs.ModePerm)

	fmt.Println(tree.String())

	return
}

type Regffs struct {
	r      io.ReadSeeker
	regf   *Regf
	header *FileHeader
}

func New(f io.ReadSeeker) *Regffs {
	regf := &Regf{}

	header := &FileHeader{}
	header.Decode(f, regf, regf)
	return &Regffs{f, regf, header}
}

func (r *Regffs) Open(name string) (fs.File, error) {
	// handleCell(int64(r.header.RootKeyOffset()+0x1000), f, r.regf)
	offset := r.header.RootKeyOffset() + 0x1000
	r.r.Seek(int64(offset), io.SeekStart)

	cell := &HiveBinCell{}
	cell.Decode(r.r, r.regf, r.regf)

	root := &File{cell: cell, r: r.r, regf: r.regf}
	if name == "." {
		return root, nil
	}
	parts := strings.Split(name, "/")
	for len(parts) > 0 {
		info, err := findInfo(root, parts[0])
		if err != nil {
			return nil, err
		}
		parts = parts[1:]
		root = info.(*File)
	}
	return root, nil
}

func findInfo(root fs.ReadDirFile, name string) (fs.DirEntry, error) {
	infos, err := root.ReadDir(0)
	if err != nil {
		return nil, err
	}
	for _, info := range infos {
		if info.Name() == name {
			return info, nil
		}
	}
	fmt.Println("not found", name)
	return nil, fs.ErrNotExist
}

type File struct {
	r         io.ReadSeeker
	cell      *HiveBinCell
	regf      *Regf
	dirOffset int
}

func (f *File) Size() int64 {
	return f.cell.CellSize()
}

func (f *File) Mode() fs.FileMode {
	if f.IsDir() {
		return fs.ModeDir
	}
	return 0
}

func (f *File) ModTime() time.Time {
	return time.Time{} // TODO
}

func (f *File) Sys() interface{} {
	return nil
}

func (f *File) Name() string {
	switch k := f.cell.Data().(type) {
	case *NamedKey:
		return string(k.UnknownString())
	case *SubKeyListVk:
		return string(k.ValueName())
	}
	panic(fmt.Sprintf("%#v", f.cell))
	return fmt.Sprintf("ERROR %T '%s'", f.cell, f.cell.Identifier())
	panic("wrong type" + string(f.cell.Identifier()))
}

func (f *File) IsDir() bool {
	return string(f.cell.Identifier()) == "nk"
}

func (f *File) Type() fs.FileMode {
	return f.Mode() & fs.ModeType
}

func (f *File) Info() (fs.FileInfo, error) {
	return f, nil
}

func (f *File) Stat() (fs.FileInfo, error) {
	return f, nil
}

func (f *File) ReadDir(n int) ([]fs.DirEntry, error) {
	if string(f.cell.Identifier()) != "nk" {
		return nil, syscall.EPERM
	}
	nk := f.cell.Data().(*NamedKey)

	var entries []fs.DirEntry
	if nk.NumberOfSubKeys() > 0 {
		cell, err := getCell(int64(nk.SubKeysListOffset())+0x1000, f.r, f.regf)
		if err == nil {
			lhlf := cell.Data().(*SubKeyListLhLf)
			for _, item := range lhlf.Items() {
				scell, err := getCell(int64(item.NamedKeyOffset())+0x1000, f.r, f.regf)
				if err != nil {
					continue
				}
				entries = append(entries, &File{r: f.r, cell: scell, regf: f.regf})
			}
		}
	}
	if nk.NumberOfValues() > 0 {
		f.r.Seek(int64(nk.ValuesListOffset())+0x1000, io.SeekStart)

		valueListOffsets := make([]uint32, nk.NumberOfValues()+1)
		binary.Read(f, binary.LittleEndian, valueListOffsets)

		for _, o := range valueListOffsets {
			if o == 0xfffffff0 {
				continue
			}
			cell, err := getCell(int64(o)+0x1000, f.r, f.regf)
			if err != nil {
				continue
			}
			entries = append(entries, &File{r: f.r, cell: cell, regf: f.regf})
		}
	}

	var err error
	if n > 0 && f.dirOffset+n > len(entries) {
		err = io.EOF
		if f.dirOffset > len(entries) {
			return nil, err
		}
	}

	if n > 0 && f.dirOffset+n <= len(entries) {
		entries = entries[f.dirOffset : f.dirOffset+n]
		f.dirOffset += n
	} else {
		entries = entries[f.dirOffset:]
		f.dirOffset += len(entries)
	}

	return entries, nil
}

func (f *File) Read(i []byte) (int, error) {
	if string(f.cell.Identifier()) != "vk" {
		return 0, syscall.EPERM
	}
	vk := f.cell.Data().(*SubKeyListVk)
	if vk.DataOffset() == 0 {
		return 0, nil
	}

	if vk.DataSize() == 0 {
		return 0, nil
	}

	isSet := vk.DataSize()&0x80000000 > 0
	dataSize := vk.DataSize()
	if isSet {
		dataSize -= 0x80000000
		data := i32tob(vk.DataOffset())
		return copy(i, data), nil
	}

	f.r.Seek(int64(vk.DataOffset())+0x1000, io.SeekStart)
	var osize int32
	binary.Read(f, binary.LittleEndian, &osize)
	size := osize
	if osize < 0 {
		size = -osize
	}
	if size < 4 {
		return 0, nil
	}
	if size > 4096 {
		return 0, nil
	}
	return f.r.Read(i)
}

func (f *File) Close() error {
	return nil
}

var dataTypes = map[uint32]string{
	1:  "RegSz",
	2:  "RegExpandSz",
	3:  "RegBinary",
	4:  "RegDword",
	7:  "RegMultiSz",
	0:  "RegNone",
	5:  "RegDwordBigEndian",
	6:  "RegLink",
	8:  "RegResourceList",
	9:  "RegFullResourceDescriptor",
	10: "RegResourceRequirementsList",
	11: "RegQword",
}

func i32tob(val uint32) []byte {
	r := make([]byte, 4)
	for i := uint32(0); i < 4; i++ {
		r[i] = byte((val >> (8 * i)) & 0xff)
	}
	return r
}

func getCell(offset int64, f io.ReadSeeker, regf *Regf) (*HiveBinCell, error) {
	f.Seek(offset, io.SeekStart)

	cell := &HiveBinCell{}
	cell.Decode(f, regf, regf)

	if bytes.Equal(cell.Identifier(), []byte{0x00, 0x00}) {
		return nil, errors.New("nope")
	}

	return cell, nil
}

func handleCell(offset int64, f io.ReadSeeker, regf *Regf, tree treeprint.Tree) {
	f.Seek(offset, io.SeekStart)

	cell := &HiveBinCell{}
	cell.Decode(f, regf, regf)

	if cell.CellSize() == 0 {
		return
	}

	switch string(cell.Identifier()) {
	case "lh":
		fallthrough
	case "lf":
		lhlf := cell.Data().(*SubKeyListLhLf)
		for _, item := range lhlf.Items() {
			handleCell(int64(item.NamedKeyOffset())+0x1000, f, regf, tree)
		}
	case "li":
		// cell.Data().(*SubKeyListLi)
	case "ri":
		// cell.Data().(*SubKeyListRi)
	case "vk":
		vk := cell.Data().(*SubKeyListVk)
		if vk.DataOffset() == 0 {
			return
		}
		var branch string
		if vk.ValueNameSize() == 0 {
			branch = "DEFAULT"
		} else if len(vk.ValueName()) > 50 {
			branch = string(vk.ValueName()[:50])
		} else {
			branch = string(vk.ValueName())
		}

		if vk.DataSize() == 0 {
			tree.AddNode(fmt.Sprintf("[%s] %s: EMPTY", dataTypes[vk.DataType()], branch))
		}

		isSet := vk.DataSize()&0x80000000 > 0
		dataSize := vk.DataSize()
		if isSet {
			dataSize -= 0x80000000
			data := i32tob(vk.DataOffset())
			s, _ := DecodeUTF16(data)
			s = strings.TrimRight(s, "\x00")
			tree.AddNode(fmt.Sprintf("[%s, %d, %t] %s: %s", dataTypes[vk.DataType()], dataSize, isSet, branch, s))
			return
		}

		switch vk.DataType() {
		case DataTypeEnum.RegSz, DataTypeEnum.RegExpandSz, DataTypeEnum.RegMultiSz:
			f.Seek(int64(vk.DataOffset())+0x1000, io.SeekStart)
			var osize int32
			binary.Read(f, binary.LittleEndian, &osize)
			size := osize
			if osize < 0 {
				size = -osize
			}
			if size < 4 {
				return
			}
			if size > 4096 {
				return
			}
			data := make([]byte, size)
			f.Read(data)
			s, _ := DecodeUTF16(data)
			parts := strings.SplitN(s, "\x00", 2)
			tree.AddNode(fmt.Sprintf("[%s, %d, %d, %t] %s: %s", dataTypes[vk.DataType()], dataSize, osize, isSet, branch, parts[0]))
		default:
			tree.AddNode(fmt.Sprintf("[%s] %s: XXX", dataTypes[vk.DataType()], branch))
		}

	case "sk":
		// cell.Data().(*SubKeyListSk)
	case "nk":
		nk := cell.Data().(*NamedKey)
		var branch string
		if len(nk.unknownString) > 50 {
			branch = string(nk.unknownString[:50])
		} else {
			branch = string(nk.unknownString)
		}
		if branch[0] == 0 {
			nk.unknownString = []byte("")
			fmt.Printf("%x: %#v\n", offset, nk)
		}

		child := tree.AddBranch(fmt.Sprintf("[%d %d %t] %s", nk.NumberOfValues(), nk.NumberOfSubKeys(), nk.Flags()&NkFlags.KeyCompName > 0, branch))

		if nk.NumberOfSubKeys() > 0 {
			handleCell(int64(nk.SubKeysListOffset())+0x1000, f, regf, child)
		}
		if nk.NumberOfValues() > 0 {
			f.Seek(int64(nk.ValuesListOffset())+0x1000, io.SeekStart)

			valueListOffsets := make([]uint32, nk.NumberOfValues()+1)
			binary.Read(f, binary.LittleEndian, valueListOffsets)

			for _, o := range valueListOffsets {
				if o == 0xfffffff0 {
					continue
				}
				handleCell(int64(o)+0x1000, f, regf, child)
			}
		}
	default:
		return
	}
	return
}

func DecodeUTF16(b []byte) (string, error) {
	if len(b)%2 != 0 {
		return "", fmt.Errorf("must have even length byte slice")
	}

	u16s := make([]uint16, 1)
	ret := &bytes.Buffer{}
	b8buf := make([]byte, 4)

	lb := len(b)
	for i := 0; i < lb; i += 2 {
		u16s[0] = uint16(b[i]) + (uint16(b[i+1]) << 8)
		r := utf16.Decode(u16s)
		n := utf8.EncodeRune(b8buf, r[0])
		ret.Write(b8buf[:n])
	}

	return ret.String(), nil
}
