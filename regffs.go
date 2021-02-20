package regffs

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"io/fs"
	"sort"
	"strings"
	"syscall"
	"time"
)

type Regffs struct {
	r      io.ReadSeeker
	regf   *Regf
	header *FileHeader
}

func New(f io.ReadSeeker) (*Regffs, error) {
	regf := &Regf{}

	header := &FileHeader{}
	err := header.Decode(f, regf, regf)
	if err != nil {
		return nil, err
	}
	return &Regffs{f, regf, header}, nil
}

func (r *Regffs) Open(name string) (fs.File, error) {
	offset := r.header.RootKeyOffset() + 0x1000
	_, err := r.r.Seek(int64(offset), io.SeekStart)
	if err != nil {
		return nil, err
	}

	cell := &HiveBinCell{}
	err = cell.Decode(r.r, r.regf, r.regf)
	if err != nil {
		return nil, err
	}

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
		return strings.Trim(string(k.UnknownString()), "\x00")
	case *SubKeyListVk:
		return strings.Trim(string(k.ValueName()), "\x00")
	}
	return "ERROR"
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
		_, err := f.r.Seek(int64(nk.ValuesListOffset())+0x1000, io.SeekStart)
		if err != nil {
			return nil, err
		}

		valueListOffsets := make([]uint32, nk.NumberOfValues()+1)
		_ = binary.Read(f, binary.LittleEndian, valueListOffsets)

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
		if f.dirOffset >= len(entries) {
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

	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
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

	_, err := f.r.Seek(int64(vk.DataOffset())+0x1000, io.SeekStart)
	if err != nil {
		return 0, err
	}

	return f.r.Read(i)
}

func (f *File) Close() error {
	return nil
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
