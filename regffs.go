package regffs

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"sort"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
	"unicode/utf8"
)

type Regffs struct {
	reader io.ReadSeeker
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
	_, err := r.reader.Seek(int64(offset), io.SeekStart)
	if err != nil {
		return nil, err
	}

	cell := &HiveBinCell{}
	err = cell.Decode(r.reader, r.regf, r.regf)
	if err != nil {
		return nil, err
	}

	root := &File{cell: cell, reader: r.reader, regf: r.regf}
	if name == "." {
		return root, nil
	}
	parts := strings.Split(name, "/")
	for i := 0; i < len(parts); i++ {
		info, err := findInfo(root, parts[i])
		if err != nil {
			return nil, err
		}
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
	reader    io.ReadSeeker
	cell      *HiveBinCell
	regf      *Regf
	dirOffset int
	data      *bytes.Reader
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
		name := string(k.ValueName())
		if name == "" {
			return "(default)"
		}
		return name
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
		entries = f.getSubkeys(int64(nk.SubKeysListOffset()) + 0x1000)
	}
	if nk.NumberOfValues() > 0 {
		valueEntries, err := f.getValues(nk)
		if err != nil {
			return nil, err
		}
		entries = append(entries, valueEntries...)
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

func (f *File) getSubkeys(offset int64) []fs.DirEntry {
	cell, err := getCell(offset, f.reader, f.regf)
	if err != nil {
		return nil
	}
	var entries []fs.DirEntry
	switch k := cell.Data().(type) {
	case *SubKeyListRi:
		for _, item := range k.Items() {
			entries = append(entries, f.getSubkeys(int64(item.SubKeyListOffset())+0x1000)...)
		}
	case *SubKeyListLhLf:
		for _, item := range k.Items() {
			entries = append(entries, f.getSubkeys(int64(item.NamedKeyOffset())+0x1000)...)
		}
	case *NamedKey:
		entries = append(entries, &File{reader: f.reader, cell: cell, regf: f.regf})
	}
	return entries
}

func (f *File) getValues(nk *NamedKey) ([]fs.DirEntry, error) {
	_, err := f.reader.Seek(int64(nk.ValuesListOffset())+0x1000, io.SeekStart)
	if err != nil {
		return nil, err
	}

	valueListOffsets := make([]uint32, nk.NumberOfValues()+1)
	_ = binary.Read(f.reader, binary.LittleEndian, valueListOffsets)

	var entries []fs.DirEntry
	for _, o := range valueListOffsets {
		if o == 0xfffffff0 {
			continue
		}
		cell, err := getCell(int64(o)+0x1000, f.reader, f.regf)
		if err != nil {
			continue
		}
		entries = append(entries, &File{reader: f.reader, cell: cell, regf: f.regf})
	}
	return entries, nil
}

func (f *File) Read(i []byte) (int, error) {
	if string(f.cell.Identifier()) != "vk" {
		return 0, syscall.EPERM
	}

	vk := f.cell.Data().(*SubKeyListVk)

	if vk.DataOffset() == 0 {
		return 0, io.EOF
	}

	if vk.DataSize() == 0 {
		return 0, io.EOF
	}

	if f.data == nil {
		err := f.loadData(vk)
		if err != nil {
			return 0, err
		}
	}

	return f.data.Read(i)
}

func (f *File) loadData(vk *SubKeyListVk) error {
	isSet := vk.DataSize()&0x80000000 > 0
	// dataSize := vk.DataSize()
	var data []byte
	if isSet {
		// dataSize -= 0x80000000
		data = i32tob(vk.DataOffset())
		// return copy(i, data), io.EOF
	} else {
		_, err := f.reader.Seek(int64(vk.DataOffset())+0x1000+4, io.SeekStart)
		if err != nil {
			return err
		}

		if vk.DataSize() > 16383*2 {
			return errors.New("entry too large")
		}
		d := make([]byte, vk.DataSize())
		_, err = io.ReadAtLeast(f.reader, d, int(vk.DataSize()))
		if err != nil {
			return err
		}
		data = d[:vk.DataSize()]
	}

	f.data = bytes.NewReader(data)
	return nil
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

func getCell(offset int64, r io.ReadSeeker, regf *Regf) (*HiveBinCell, error) {
	_, err := r.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, err
	}

	cell := &HiveBinCell{}
	err = cell.Decode(r, regf, regf)
	if err != nil {
		return nil, err
	}

	if bytes.Equal(cell.Identifier(), []byte{0x00, 0x00}) {
		return nil, errors.New("invalid cell")
	}

	return cell, nil
}

func DecodeRegSz(b []byte) (string, error) {
	s, err := DecodeUTF16(b)
	if err != nil {
		return "", err
	}
	return s[:len(s)-1], err
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
