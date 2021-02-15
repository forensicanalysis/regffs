// file generated at 2021-02-14T13:42:30Z

package main

import (
	"encoding/binary"
	"io"
)

type KSYDecoder interface {
	Decode(io.ReadSeeker, ...interface{}) error
}


/* This spec allows to parse files used by Microsoft Windows family of
operating systems to store parts of its "registry". "Registry" is a
hierarchical database that is used to store system settings (global
configuration, per-user, per-application configuration, etc).

Typically, registry files are stored in:

* System-wide: several files in `%SystemRoot%\System32\Config\`
* User-wide:
  * `%USERPROFILE%\Ntuser.dat`
  * `%USERPROFILE%\Local Settings\Application Data\Microsoft\Windows\Usrclass.dat` (localized, Windows 2000, Server 2003 and Windows XP)
  * `%USERPROFILE%\AppData\Local\Microsoft\Windows\Usrclass.dat` (non-localized, Windows Vista and later)

Note that one typically can't access files directly on a mounted
filesystem with a running Windows OS.*/
type Regf struct {
	decoder  io.ReadSeeker
	parent   interface{}
	root     interface{}
	header   *FileHeader `ks:"header,attribute"`
	hiveBins []HiveBin   `ks:"hive_bins,attribute"`
}

func (k *Regf) Parent() *Regf {
	return k.parent.(*Regf)
}
func (k *Regf) Root() *Regf {
	return k.root.(*Regf)
}
func (k *Regf) Decode(reader io.ReadSeeker, ancestors ...interface{}) (err error) {
	if k.decoder == nil {
		if reader == nil {
			panic("Neither k.decoder nor reader are set.")
		}
		k.decoder = reader
	}
	if len(ancestors) == 2 {
		k.parent = ancestors[0]
		k.root = ancestors[1]
	} else if len(ancestors) == 0 {
		k.parent = k
		k.root = k
	} else {
		panic("To many ancestors are given.")
	}
	if err == nil {
		var elem FileHeader
		err = elem.Decode(k.decoder, k, k.Root())
		k.header = &elem
	}
	if err == nil {
		var elem HiveBin
		k.hiveBins = []HiveBin{}
		for index := 0; true; index++ {
			// fmt.Println("decode bin", index) // TODO
			pos, _ := k.decoder.Seek(0, io.SeekCurrent)
			err = elem.Decode(k.decoder, k, k.Root())
			if elem.Header().Size() == 0 {
				break
			}
			pos = pos + int64(elem.Header().Size())
			_, err = k.decoder.Seek(pos, io.SeekStart)
			if err != nil {
				if err == io.EOF {
					err = nil
				}
				break
			}
			k.hiveBins = append(k.hiveBins, elem)
		}
	}
	return
}
func (k *Regf) Header() (value *FileHeader) {
	return k.header
}
func (k *Regf) HiveBins() (value []HiveBin) {
	return k.hiveBins
}

type HiveBinCell struct {
	decoder        io.ReadSeeker
	parent         interface{}
	root           interface{}
	cellSizeRaw    int32              `ks:"cell_size_raw,attribute"`
	identifier     []byte             `ks:"identifier,attribute"`
	data           KSYDecoder `ks:"data,attribute"`
	cellSize       int64              `ks:"cell_size,instance"`
	cellSizeSet    bool
	isAllocated    bool `ks:"is_allocated,instance"`
	isAllocatedSet bool
}

func (k *HiveBinCell) Parent() *HiveBin {
	return k.parent.(*HiveBin)
}
func (k *HiveBinCell) Root() *Regf {
	return k.root.(*Regf)
}
func (k *HiveBinCell) Decode(reader io.ReadSeeker, ancestors ...interface{}) (err error) {
	if k.decoder == nil {
		if reader == nil {
			panic("Neither k.decoder nor reader are set.")
		}
		k.decoder = reader
	}
	if len(ancestors) == 2 {
		k.parent = ancestors[0]
		k.root = ancestors[1]
	} else if len(ancestors) == 0 {
		k.parent = k
		k.root = k
	} else {
		panic("To many ancestors are given.")
	}
	if err == nil {
		var elem int32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.cellSizeRaw = elem
	}
	if err == nil {
		var elem []byte
		elem = make([]byte, 2)
		pos, _ := k.decoder.Seek(0, io.SeekCurrent)
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		pos = pos + int64(2)
		_, err = k.decoder.Seek(pos, io.SeekStart)
		k.identifier = elem
	}
	if err == nil {
		var elem KSYDecoder
		switch string(k.Identifier()) {
		case "lh":
			elem = &SubKeyListLhLf{}
		case "lf":
			elem = &SubKeyListLhLf{}
		case "li":
			elem = &SubKeyListLi{}
		case "ri":
			elem = &SubKeyListRi{}
		case "vk":
			elem = &SubKeyListVk{}
		case "sk":
			elem = &SubKeyListSk{}
		case "nk":
			elem = &NamedKey{}
		default:
			return
			// return fmt.Errorf("unknown identifier %s", k.Identifier())
		}
		pos, _ := k.decoder.Seek(0, io.SeekCurrent)
		err = elem.Decode(k.decoder, k, k.Root())
		pos = pos + int64(k.CellSize()-2-4)
		_, err = k.decoder.Seek(pos, io.SeekStart)
		k.data = elem
	}
	return
}

func (k *HiveBinCell) CellSizeRaw() (value int32) {
	return k.cellSizeRaw
}
func (k *HiveBinCell) Identifier() (value []byte) {
	return k.identifier
}
func (k *HiveBinCell) Data() (value KSYDecoder) {
	return k.data
}
func (k *HiveBinCell) CellSize() (value int64) {
	// if !k.cellSizeSet {
	k.cellSize = func() int64 {
		if k.CellSizeRaw() < 0 {
			return int64(-1 * k.CellSizeRaw())
		} else {
			return int64(k.CellSizeRaw())
		}
	}()
	k.cellSizeSet = true
	// }
	return k.cellSize
}
func (k *HiveBinCell) IsAllocated() (value bool) {
	if !k.isAllocatedSet {
		k.isAllocated = k.CellSizeRaw() < 0
		k.isAllocatedSet = true
	}
	return k.isAllocated
}

type SubKeyListLhLf struct {
	decoder io.ReadSeeker
	parent  interface{}
	root    interface{}
	count   uint16     `ks:"count,attribute"`
	items   []LhLfItem `ks:"items,attribute"`
}

func (k *SubKeyListLhLf) Parent() *HiveBinCell {
	return k.parent.(*HiveBinCell)
}
func (k *SubKeyListLhLf) Root() *Regf {
	return k.root.(*Regf)
}
func (k *SubKeyListLhLf) Decode(reader io.ReadSeeker, ancestors ...interface{}) (err error) {
	if k.decoder == nil {
		if reader == nil {
			panic("Neither k.decoder nor reader are set.")
		}
		k.decoder = reader
	}
	if len(ancestors) == 2 {
		k.parent = ancestors[0]
		k.root = ancestors[1]
	} else if len(ancestors) == 0 {
		k.parent = k
		k.root = k
	} else {
		panic("To many ancestors are given.")
	}
	if err == nil {
		var elem uint16
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.count = elem
	}
	if err == nil {
		var elem LhLfItem
		k.items = []LhLfItem{}
		for index := 0; index < int(k.Count()); index++ {
			err = elem.Decode(k.decoder, k, k.Root())
			if err != nil {
				if err == io.EOF {
					err = nil
				}
				break
			}
			k.items = append(k.items, elem)
		}
	}
	return
}
func (k *SubKeyListLhLf) Count() (value uint16) {
	return k.count
}
func (k *SubKeyListLhLf) Items() (value []LhLfItem) {
	return k.items
}

type LhLfItem struct {
	decoder        io.ReadSeeker
	parent         interface{}
	root           interface{}
	namedKeyOffset uint32 `ks:"named_key_offset,attribute"`
	hashValue      uint32 `ks:"hash_value,attribute"`
}

func (k *LhLfItem) Parent() *SubKeyListLhLf {
	return k.parent.(*SubKeyListLhLf)
}
func (k *LhLfItem) Root() *Regf {
	return k.root.(*Regf)
}
func (k *LhLfItem) Decode(reader io.ReadSeeker, ancestors ...interface{}) (err error) {
	if k.decoder == nil {
		if reader == nil {
			panic("Neither k.decoder nor reader are set.")
		}
		k.decoder = reader
	}
	if len(ancestors) == 2 {
		k.parent = ancestors[0]
		k.root = ancestors[1]
	} else if len(ancestors) == 0 {
		k.parent = k
		k.root = k
	} else {
		panic("To many ancestors are given.")
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.namedKeyOffset = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.hashValue = elem
	}
	return
}
func (k *LhLfItem) NamedKeyOffset() (value uint32) {
	return k.namedKeyOffset
}
func (k *LhLfItem) HashValue() (value uint32) {
	return k.hashValue
}

type SubKeyListLi struct {
	decoder io.ReadSeeker
	parent  interface{}
	root    interface{}
	count   uint16   `ks:"count,attribute"`
	items   []LiItem `ks:"items,attribute"`
}

func (k *SubKeyListLi) Parent() *HiveBinCell {
	return k.parent.(*HiveBinCell)
}
func (k *SubKeyListLi) Root() *Regf {
	return k.root.(*Regf)
}
func (k *SubKeyListLi) Decode(reader io.ReadSeeker, ancestors ...interface{}) (err error) {
	if k.decoder == nil {
		if reader == nil {
			panic("Neither k.decoder nor reader are set.")
		}
		k.decoder = reader
	}
	if len(ancestors) == 2 {
		k.parent = ancestors[0]
		k.root = ancestors[1]
	} else if len(ancestors) == 0 {
		k.parent = k
		k.root = k
	} else {
		panic("To many ancestors are given.")
	}
	if err == nil {
		var elem uint16
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.count = elem
	}
	if err == nil {
		var elem LiItem
		k.items = []LiItem{}
		for index := 0; index < int(k.Count()); index++ {
			err = elem.Decode(k.decoder, k, k.Root())
			if err != nil {
				if err == io.EOF {
					err = nil
				}
				break
			}
			k.items = append(k.items, elem)
		}
	}
	return
}
func (k *SubKeyListLi) Count() (value uint16) {
	return k.count
}
func (k *SubKeyListLi) Items() (value []LiItem) {
	return k.items
}

type LiItem struct {
	decoder        io.ReadSeeker
	parent         interface{}
	root           interface{}
	namedKeyOffset uint32 `ks:"named_key_offset,attribute"`
}

func (k *LiItem) Parent() *SubKeyListLi {
	return k.parent.(*SubKeyListLi)
}
func (k *LiItem) Root() *Regf {
	return k.root.(*Regf)
}
func (k *LiItem) Decode(reader io.ReadSeeker, ancestors ...interface{}) (err error) {
	if k.decoder == nil {
		if reader == nil {
			panic("Neither k.decoder nor reader are set.")
		}
		k.decoder = reader
	}
	if len(ancestors) == 2 {
		k.parent = ancestors[0]
		k.root = ancestors[1]
	} else if len(ancestors) == 0 {
		k.parent = k
		k.root = k
	} else {
		panic("To many ancestors are given.")
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.namedKeyOffset = elem
	}
	return
}
func (k *LiItem) NamedKeyOffset() (value uint32) {
	return k.namedKeyOffset
}

type SubKeyListRi struct {
	decoder io.ReadSeeker
	parent  interface{}
	root    interface{}
	count   uint16   `ks:"count,attribute"`
	items   []RiItem `ks:"items,attribute"`
}

func (k *SubKeyListRi) Parent() *HiveBinCell {
	return k.parent.(*HiveBinCell)
}
func (k *SubKeyListRi) Root() *Regf {
	return k.root.(*Regf)
}
func (k *SubKeyListRi) Decode(reader io.ReadSeeker, ancestors ...interface{}) (err error) {
	if k.decoder == nil {
		if reader == nil {
			panic("Neither k.decoder nor reader are set.")
		}
		k.decoder = reader
	}
	if len(ancestors) == 2 {
		k.parent = ancestors[0]
		k.root = ancestors[1]
	} else if len(ancestors) == 0 {
		k.parent = k
		k.root = k
	} else {
		panic("To many ancestors are given.")
	}
	if err == nil {
		var elem uint16
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.count = elem
	}
	if err == nil {
		var elem RiItem
		k.items = []RiItem{}
		for index := 0; index < int(k.Count()); index++ {
			err = elem.Decode(k.decoder, k, k.Root())
			if err != nil {
				if err == io.EOF {
					err = nil
				}
				break
			}
			k.items = append(k.items, elem)
		}
	}
	return
}
func (k *SubKeyListRi) Count() (value uint16) {
	return k.count
}
func (k *SubKeyListRi) Items() (value []RiItem) {
	return k.items
}

type RiItem struct {
	decoder          io.ReadSeeker
	parent           interface{}
	root             interface{}
	subKeyListOffset uint32 `ks:"sub_key_list_offset,attribute"`
}

func (k *RiItem) Parent() *SubKeyListRi {
	return k.parent.(*SubKeyListRi)
}
func (k *RiItem) Root() *Regf {
	return k.root.(*Regf)
}
func (k *RiItem) Decode(reader io.ReadSeeker, ancestors ...interface{}) (err error) {
	if k.decoder == nil {
		if reader == nil {
			panic("Neither k.decoder nor reader are set.")
		}
		k.decoder = reader
	}
	if len(ancestors) == 2 {
		k.parent = ancestors[0]
		k.root = ancestors[1]
	} else if len(ancestors) == 0 {
		k.parent = k
		k.root = k
	} else {
		panic("To many ancestors are given.")
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.subKeyListOffset = elem
	}
	return
}
func (k *RiItem) SubKeyListOffset() (value uint32) {
	return k.subKeyListOffset
}

type SubKeyListVk struct {
	decoder       io.ReadSeeker
	parent        interface{}
	root          interface{}
	valueNameSize uint16 `ks:"value_name_size,attribute"`
	dataSize      uint32 `ks:"data_size,attribute"`
	dataOffset    uint32 `ks:"data_offset,attribute"`
	dataType      uint32 `ks:"data_type,attribute"`
	flags         uint16 `ks:"flags,attribute"`
	padding       uint16 `ks:"padding,attribute"`
	valueName     []byte `ks:"value_name,attribute"`
}

func (k *SubKeyListVk) Parent() *HiveBinCell {
	return k.parent.(*HiveBinCell)
}
func (k *SubKeyListVk) Root() *Regf {
	return k.root.(*Regf)
}
func (k *SubKeyListVk) Decode(reader io.ReadSeeker, ancestors ...interface{}) (err error) {
	if k.decoder == nil {
		if reader == nil {
			panic("Neither k.decoder nor reader are set.")
		}
		k.decoder = reader
	}
	if len(ancestors) == 2 {
		k.parent = ancestors[0]
		k.root = ancestors[1]
	} else if len(ancestors) == 0 {
		k.parent = k
		k.root = k
	} else {
		panic("To many ancestors are given.")
	}
	if err == nil {
		var elem uint16
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.valueNameSize = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.dataSize = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.dataOffset = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.dataType = elem
	}
	if err == nil {
		var elem uint16
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.flags = elem
	}
	if err == nil {
		var elem uint16
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.padding = elem
	}
	if err == nil {
		var elem []byte
		if k.Flags() == VkFlags.ValueCompName {
			elem = make([]byte, k.ValueNameSize())
			pos, _ := k.decoder.Seek(0, io.SeekCurrent)
			err = binary.Read(k.decoder, binary.LittleEndian, &elem)
			pos = pos + int64(k.ValueNameSize())
			_, err = k.decoder.Seek(pos, io.SeekStart)
			k.valueName = elem
		}
	}
	return
}
func (k *SubKeyListVk) ValueNameSize() (value uint16) {
	return k.valueNameSize
}
func (k *SubKeyListVk) DataSize() (value uint32) {
	return k.dataSize
}
func (k *SubKeyListVk) DataOffset() (value uint32) {
	return k.dataOffset
}
func (k *SubKeyListVk) DataType() (value uint32) {
	return k.dataType
}
func (k *SubKeyListVk) Flags() (value uint16) {
	return k.flags
}
func (k *SubKeyListVk) Padding() (value uint16) {
	return k.padding
}
func (k *SubKeyListVk) ValueName() (value []byte) {
	return k.valueName
}

var DataTypeEnum = struct {
	RegDwordBigEndian           uint32
	RegLink                     uint32
	RegResourceList             uint32
	RegFullResourceDescriptor   uint32
	RegResourceRequirementsList uint32
	RegQword                    uint32
	RegNone                     uint32
	RegExpandSz                 uint32
	RegBinary                   uint32
	RegDword                    uint32
	RegMultiSz                  uint32
	RegSz                       uint32
}{
	RegSz:                       1,
	RegExpandSz:                 2,
	RegBinary:                   3,
	RegDword:                    4,
	RegMultiSz:                  7,
	RegNone:                     0,
	RegDwordBigEndian:           5,
	RegLink:                     6,
	RegResourceList:             8,
	RegFullResourceDescriptor:   9,
	RegResourceRequirementsList: 10,
	RegQword:                    11,
}
var VkFlags = struct {
	ValueCompName uint16
}{
	ValueCompName: 1,
}

type SubKeyListSk struct {
	decoder                   io.ReadSeeker
	parent                    interface{}
	root                      interface{}
	unknown1                  uint16 `ks:"unknown1,attribute"`
	previousSecurityKeyOffset uint32 `ks:"previous_security_key_offset,attribute"`
	nextSecurityKeyOffset     uint32 `ks:"next_security_key_offset,attribute"`
	referenceCount            uint32 `ks:"reference_count,attribute"`
}

func (k *SubKeyListSk) Parent() *HiveBinCell {
	return k.parent.(*HiveBinCell)
}
func (k *SubKeyListSk) Root() *Regf {
	return k.root.(*Regf)
}
func (k *SubKeyListSk) Decode(reader io.ReadSeeker, ancestors ...interface{}) (err error) {
	if k.decoder == nil {
		if reader == nil {
			panic("Neither k.decoder nor reader are set.")
		}
		k.decoder = reader
	}
	if len(ancestors) == 2 {
		k.parent = ancestors[0]
		k.root = ancestors[1]
	} else if len(ancestors) == 0 {
		k.parent = k
		k.root = k
	} else {
		panic("To many ancestors are given.")
	}
	if err == nil {
		var elem uint16
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.unknown1 = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.previousSecurityKeyOffset = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.nextSecurityKeyOffset = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.referenceCount = elem
	}
	return
}
func (k *SubKeyListSk) Unknown1() (value uint16) {
	return k.unknown1
}
func (k *SubKeyListSk) PreviousSecurityKeyOffset() (value uint32) {
	return k.previousSecurityKeyOffset
}
func (k *SubKeyListSk) NextSecurityKeyOffset() (value uint32) {
	return k.nextSecurityKeyOffset
}
func (k *SubKeyListSk) ReferenceCount() (value uint32) {
	return k.referenceCount
}

type NamedKey struct {
	decoder                    io.ReadSeeker
	parent                     interface{}
	root                       interface{}
	flags                      uint16    `ks:"flags,attribute"`
	lastKeyWrittenDateAndTime  *Filetime `ks:"last_key_written_date_and_time,attribute"`
	unknown1                   uint32    `ks:"unknown1,attribute"`
	parentKeyOffset            uint32    `ks:"parent_key_offset,attribute"`
	numberOfSubKeys            uint32    `ks:"number_of_sub_keys,attribute"`
	numberOfVolatileSubKeys    uint32    `ks:"number_of_volatile_sub_keys,attribute"`
	subKeysListOffset          uint32    `ks:"sub_keys_list_offset,attribute"`
	volatileSubKeysListOffset  uint32    `ks:"volatile_sub_keys_list_offset,attribute"`
	numberOfValues             uint32    `ks:"number_of_values,attribute"`
	valuesListOffset           uint32    `ks:"values_list_offset,attribute"`
	securityKeyOffset          uint32    `ks:"security_key_offset,attribute"`
	classNameOffset            uint32    `ks:"class_name_offset,attribute"`
	largestSubKeyNameSize      uint32    `ks:"largest_sub_key_name_size,attribute"`
	largestSubKeyClassNameSize uint32    `ks:"largest_sub_key_class_name_size,attribute"`
	largestValueNameSize       uint32    `ks:"largest_value_name_size,attribute"`
	largestValueDataSize       uint32    `ks:"largest_value_data_size,attribute"`
	unknown2                   uint32    `ks:"unknown2,attribute"`
	keyNameSize                uint16    `ks:"key_name_size,attribute"`
	classNameSize              uint16    `ks:"class_name_size,attribute"`
	unknownStringSize          uint16    `ks:"unknown_string_size,attribute"`
	unknownStringSize2         uint16    `ks:"unknown_string_size,attribute"`
	unknownString              []byte    `ks:"unknown_string,attribute"`
}

func (k *NamedKey) Parent() *HiveBinCell {
	return k.parent.(*HiveBinCell)
}
func (k *NamedKey) Root() *Regf {
	return k.root.(*Regf)
}
func (k *NamedKey) Decode(reader io.ReadSeeker, ancestors ...interface{}) (err error) {
	if k.decoder == nil {
		if reader == nil {
			panic("Neither k.decoder nor reader are set.")
		}
		k.decoder = reader
	}
	if len(ancestors) == 2 {
		k.parent = ancestors[0]
		k.root = ancestors[1]
	} else if len(ancestors) == 0 {
		k.parent = k
		k.root = k
	} else {
		panic("To many ancestors are given.")
	}
	if err == nil {
		var elem uint16
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.flags = elem
	}
	if err == nil {
		var elem Filetime
		err = elem.Decode(k.decoder, k, k.Root())
		k.lastKeyWrittenDateAndTime = &elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.unknown1 = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.parentKeyOffset = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.numberOfSubKeys = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.numberOfVolatileSubKeys = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.subKeysListOffset = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.volatileSubKeysListOffset = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.numberOfValues = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.valuesListOffset = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.securityKeyOffset = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.classNameOffset = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.largestSubKeyNameSize = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.largestSubKeyClassNameSize = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.largestValueNameSize = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.largestValueDataSize = elem
	}
	//if err == nil {
	//	var elem uint32
	//	err = binary.Read(k.decoder, binary.LittleEndian, &elem)
	//	k.unknown2 = elem
	//}
	if err == nil {
		var elem uint16
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.keyNameSize = elem
	}
	if err == nil {
		var elem uint16
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.classNameSize = elem
	}
	if err == nil {
		var elem uint16
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.unknownStringSize = elem
	}
	if err == nil {
		var elem uint16
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.unknownStringSize2 = elem
	}
	if err == nil {
		var elem []byte
		elem = make([]byte, k.UnknownStringSize())
		pos, _ := k.decoder.Seek(0, io.SeekCurrent)
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		pos = pos + int64(k.UnknownStringSize())
		_, err = k.decoder.Seek(pos, io.SeekStart)
		k.unknownString = elem
	}
	return
}
func (k *NamedKey) Flags() (value uint16) {
	return k.flags
}
func (k *NamedKey) LastKeyWrittenDateAndTime() (value *Filetime) {
	return k.lastKeyWrittenDateAndTime
}
func (k *NamedKey) Unknown1() (value uint32) {
	return k.unknown1
}
func (k *NamedKey) ParentKeyOffset() (value uint32) {
	return k.parentKeyOffset
}
func (k *NamedKey) NumberOfSubKeys() (value uint32) {
	return k.numberOfSubKeys
}
func (k *NamedKey) NumberOfVolatileSubKeys() (value uint32) {
	return k.numberOfVolatileSubKeys
}
func (k *NamedKey) SubKeysListOffset() (value uint32) {
	return k.subKeysListOffset
}
func (k *NamedKey) NumberOfValues() (value uint32) {
	return k.numberOfValues
}
func (k *NamedKey) ValuesListOffset() (value uint32) {
	return k.valuesListOffset
}
func (k *NamedKey) SecurityKeyOffset() (value uint32) {
	return k.securityKeyOffset
}
func (k *NamedKey) ClassNameOffset() (value uint32) {
	return k.classNameOffset
}
func (k *NamedKey) LargestSubKeyNameSize() (value uint32) {
	return k.largestSubKeyNameSize
}
func (k *NamedKey) LargestSubKeyClassNameSize() (value uint32) {
	return k.largestSubKeyClassNameSize
}
func (k *NamedKey) LargestValueNameSize() (value uint32) {
	return k.largestValueNameSize
}
func (k *NamedKey) LargestValueDataSize() (value uint32) {
	return k.largestValueDataSize
}
func (k *NamedKey) Unknown2() (value uint32) {
	return k.unknown2
}
func (k *NamedKey) KeyNameSize() (value uint16) {
	return k.keyNameSize
}
func (k *NamedKey) ClassNameSize() (value uint16) {
	return k.classNameSize
}
func (k *NamedKey) UnknownStringSize() (value uint16) {
	return k.unknownStringSize
}
func (k *NamedKey) UnknownString() (value []byte) {
	return k.unknownString
}

var NkFlags = struct {
	Unknown1        uint16
	KeyHiveExit     uint16
	KeyHiveEntry    uint16
	KeyNoDelete     uint16
	KeyPrefefHandle uint16
	KeyVirtMirrored uint16
	KeyVirtTarget   uint16
	KeyVirtualStore uint16
	KeyIsVolatile   uint16
	KeySymLink      uint16
	KeyCompName     uint16
	Unknown2        uint16
}{
	KeyCompName:     32,
	Unknown2:        16384,
	KeyIsVolatile:   1,
	KeySymLink:      16,
	KeyNoDelete:     8,
	KeyPrefefHandle: 64,
	KeyVirtMirrored: 128,
	KeyVirtTarget:   256,
	KeyVirtualStore: 512,
	Unknown1:        4096,
	KeyHiveExit:     2,
	KeyHiveEntry:    4,
}

type HiveBin struct {
	decoder io.ReadSeeker
	parent  interface{}
	root    interface{}
	header  *HiveBinHeader `ks:"header,attribute"`
	cells   []HiveBinCell  `ks:"cells,attribute"`
}

func (k *HiveBin) Parent() *Regf {
	return k.parent.(*Regf)
}
func (k *HiveBin) Root() *Regf {
	return k.root.(*Regf)
}
func (k *HiveBin) Decode(reader io.ReadSeeker, ancestors ...interface{}) (err error) {
	if k.decoder == nil {
		if reader == nil {
			panic("Neither k.decoder nor reader are set.")
		}
		k.decoder = reader
	}
	if len(ancestors) == 2 {
		k.parent = ancestors[0]
		k.root = ancestors[1]
	} else if len(ancestors) == 0 {
		k.parent = k
		k.root = k
	} else {
		panic("To many ancestors are given.")
	}
	if err == nil {
		var elem HiveBinHeader
		err = elem.Decode(k.decoder, k, k.Root())
		k.header = &elem
	}
	if err == nil {
		var elem HiveBinCell
		k.cells = []HiveBinCell{}
		for index := 0; true; index++ {
			// fmt.Println("decode cell", index) // TODO
			err = elem.Decode(k.decoder, k, k.Root())
			if err != nil {
				if err == io.EOF {
					err = nil
				}
				break
			}
			k.cells = append(k.cells, elem)
		}
	}
	return
}
func (k *HiveBin) Header() (value *HiveBinHeader) {
	return k.header
}
func (k *HiveBin) Cells() (value []HiveBinCell) {
	return k.cells
}

type Filetime struct {
	decoder io.ReadSeeker
	parent  interface{}
	root    interface{}
	value   uint64 `ks:"value,attribute"`
}

func (k *Filetime) Parent() *FileHeader {
	return k.parent.(*FileHeader)
}
func (k *Filetime) Root() *Regf {
	return k.root.(*Regf)
}
func (k *Filetime) Decode(reader io.ReadSeeker, ancestors ...interface{}) (err error) {
	if k.decoder == nil {
		if reader == nil {
			panic("Neither k.decoder nor reader are set.")
		}
		k.decoder = reader
	}
	if len(ancestors) == 2 {
		k.parent = ancestors[0]
		k.root = ancestors[1]
	} else if len(ancestors) == 0 {
		k.parent = k
		k.root = k
	} else {
		panic("To many ancestors are given.")
	}
	if err == nil {
		var elem uint64
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.value = elem
	}
	return
}
func (k *Filetime) Value() (value uint64) {
	return k.value
}

type FileHeader struct {
	decoder                     io.ReadSeeker
	parent                      interface{}
	root                        interface{}
	signature                   []byte    `ks:"signature,attribute"`
	primarySequenceNumber       uint32    `ks:"primary_sequence_number,attribute"`
	secondarySequenceNumber     uint32    `ks:"secondary_sequence_number,attribute"`
	lastModificationDateAndTime *Filetime `ks:"last_modification_date_and_time,attribute"`
	majorVersion                uint32    `ks:"major_version,attribute"`
	minorVersion                uint32    `ks:"minor_version,attribute"`
	headerType                  uint32    `ks:"header_type,attribute"`
	format                      uint32    `ks:"format,attribute"`
	rootKeyOffset               uint32    `ks:"root_key_offset,attribute"`
	hiveBinsDataSize            uint32    `ks:"hive_bins_data_size,attribute"`
	clusteringFactor            uint32    `ks:"clustering_factor,attribute"`
	unknown1                    []byte    `ks:"unknown1,attribute"`
	unknown2                    []byte    `ks:"unknown2,attribute"`
	checksum                    uint32    `ks:"checksum,attribute"`
	reserved                    []byte    `ks:"reserved,attribute"`
	bootType                    uint32    `ks:"boot_type,attribute"`
	bootRecover                 uint32    `ks:"boot_recover,attribute"`
}

func (k *FileHeader) Parent() *Regf {
	return k.parent.(*Regf)
}
func (k *FileHeader) Root() *Regf {
	return k.root.(*Regf)
}
func (k *FileHeader) Decode(reader io.ReadSeeker, ancestors ...interface{}) (err error) {
	if k.decoder == nil {
		if reader == nil {
			panic("Neither k.decoder nor reader are set.")
		}
		k.decoder = reader
	}
	if len(ancestors) == 2 {
		k.parent = ancestors[0]
		k.root = ancestors[1]
	} else if len(ancestors) == 0 {
		k.parent = k
		k.root = k
	} else {
		panic("To many ancestors are given.")
	}
	if err == nil {
		var elem []byte
		elem = make([]byte, 4)
		pos, _ := k.decoder.Seek(0, io.SeekCurrent)
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		pos = pos + int64(4)
		_, err = k.decoder.Seek(pos, io.SeekStart)
		k.signature = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.primarySequenceNumber = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.secondarySequenceNumber = elem
	}
	if err == nil {
		var elem Filetime
		err = elem.Decode(k.decoder, k, k.Root())
		k.lastModificationDateAndTime = &elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.majorVersion = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.minorVersion = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.headerType = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.format = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.rootKeyOffset = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.hiveBinsDataSize = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.clusteringFactor = elem
	}
	if err == nil {
		var elem []byte
		elem = make([]byte, 64)
		pos, _ := k.decoder.Seek(0, io.SeekCurrent)
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		pos = pos + int64(64)
		_, err = k.decoder.Seek(pos, io.SeekStart)
		k.unknown1 = elem
	}
	if err == nil {
		var elem []byte
		elem = make([]byte, 396)
		pos, _ := k.decoder.Seek(0, io.SeekCurrent)
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		pos = pos + int64(396)
		_, err = k.decoder.Seek(pos, io.SeekStart)
		k.unknown2 = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.checksum = elem
	}
	if err == nil {
		var elem []byte
		elem = make([]byte, 3576)
		pos, _ := k.decoder.Seek(0, io.SeekCurrent)
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		pos = pos + int64(3576)
		_, err = k.decoder.Seek(pos, io.SeekStart)
		k.reserved = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.bootType = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.bootRecover = elem
	}
	return
}
func (k *FileHeader) Signature() (value []byte) {
	return k.signature
}
func (k *FileHeader) PrimarySequenceNumber() (value uint32) {
	return k.primarySequenceNumber
}
func (k *FileHeader) SecondarySequenceNumber() (value uint32) {
	return k.secondarySequenceNumber
}
func (k *FileHeader) LastModificationDateAndTime() (value *Filetime) {
	return k.lastModificationDateAndTime
}
func (k *FileHeader) MajorVersion() (value uint32) {
	return k.majorVersion
}
func (k *FileHeader) MinorVersion() (value uint32) {
	return k.minorVersion
}
func (k *FileHeader) HeaderType() (value uint32) {
	return k.headerType
}
func (k *FileHeader) Format() (value uint32) {
	return k.format
}
func (k *FileHeader) RootKeyOffset() (value uint32) {
	return k.rootKeyOffset
}
func (k *FileHeader) HiveBinsDataSize() (value uint32) {
	return k.hiveBinsDataSize
}
func (k *FileHeader) ClusteringFactor() (value uint32) {
	return k.clusteringFactor
}
func (k *FileHeader) Unknown1() (value []byte) {
	return k.unknown1
}
func (k *FileHeader) Unknown2() (value []byte) {
	return k.unknown2
}
func (k *FileHeader) Checksum() (value uint32) {
	return k.checksum
}
func (k *FileHeader) Reserved() (value []byte) {
	return k.reserved
}
func (k *FileHeader) BootType() (value uint32) {
	return k.bootType
}
func (k *FileHeader) BootRecover() (value uint32) {
	return k.bootRecover
}

var FileType = struct {
	TransactionLog uint32
	Normal         uint32
}{
	Normal:         0,
	TransactionLog: 1,
}
var FileFormat = struct {
	DirectMemoryLoad uint32
}{
	DirectMemoryLoad: 1,
}

type HiveBinHeader struct {
	decoder   io.ReadSeeker
	parent    interface{}
	root      interface{}
	signature []byte `ks:"signature,attribute"`
	offset    uint32 `ks:"offset,attribute"`
	/* The offset of the hive bin, Value in bytes and relative from
	   the start of the hive bin data*/
	size uint32 `ks:"size,attribute"`
	/* Size of the hive bin*/
	unknown1 uint32 `ks:"unknown1,attribute"`
	/* 0 most of the time, can contain remnant data*/
	unknown2 uint32 `ks:"unknown2,attribute"`
	/* 0 most of the time, can contain remnant data*/
	timestamp *Filetime `ks:"timestamp,attribute"`
	/* Only the root (first) hive bin seems to contain a valid FILETIME*/
	unknown4 uint32 `ks:"unknown4,attribute"`
	/* Contains number of bytes*/
}

func (k *HiveBinHeader) Parent() *HiveBin {
	return k.parent.(*HiveBin)
}
func (k *HiveBinHeader) Root() *Regf {
	return k.root.(*Regf)
}
func (k *HiveBinHeader) Decode(reader io.ReadSeeker, ancestors ...interface{}) (err error) {
	if k.decoder == nil {
		if reader == nil {
			panic("Neither k.decoder nor reader are set.")
		}
		k.decoder = reader
	}
	if len(ancestors) == 2 {
		k.parent = ancestors[0]
		k.root = ancestors[1]
	} else if len(ancestors) == 0 {
		k.parent = k
		k.root = k
	} else {
		panic("To many ancestors are given.")
	}
	if err == nil {
		var elem []byte
		elem = make([]byte, 4)
		pos, _ := k.decoder.Seek(0, io.SeekCurrent)
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		pos = pos + int64(4)
		_, err = k.decoder.Seek(pos, io.SeekStart)
		k.signature = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.offset = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.size = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.unknown1 = elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.unknown2 = elem
	}
	if err == nil {
		var elem Filetime
		err = elem.Decode(k.decoder, k, k.Root())
		k.timestamp = &elem
	}
	if err == nil {
		var elem uint32
		err = binary.Read(k.decoder, binary.LittleEndian, &elem)
		k.unknown4 = elem
	}
	return
}
func (k *HiveBinHeader) Signature() (value []byte) {
	return k.signature
}
func (k *HiveBinHeader) Offset() (value uint32) {
	return k.offset
}
func (k *HiveBinHeader) Size() (value uint32) {
	return k.size
}
func (k *HiveBinHeader) Unknown1() (value uint32) {
	return k.unknown1
}
func (k *HiveBinHeader) Unknown2() (value uint32) {
	return k.unknown2
}
func (k *HiveBinHeader) Timestamp() (value *Filetime) {
	return k.timestamp
}
func (k *HiveBinHeader) Unknown4() (value uint32) {
	return k.unknown4
}
