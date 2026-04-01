package tablebin

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
)

// Decoder 解析 EncodeFile 写出的表二进制（可复用读多行）。
type Decoder struct {
	strs  []string
	buf   []byte
	off   int
	nRows uint64
}

// Open 读取整个文件并解析文件头与字符串池；随后可按列顺序调用 Read* 读取各行。
func Open(path string) (*Decoder, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return NewDecoder(b)
}

// NewDecoder 从内存切片解析（与 Open 相同布局）。
func NewDecoder(b []byte) (*Decoder, error) {
	if len(b) < 4+1 {
		return nil, errCorrupt
	}
	if b[0] != magic0 || b[1] != magic1 || b[2] != magic2 {
		return nil, errCorrupt
	}
	if b[3] != ver {
		return nil, errVersion
	}
	off := 4
	nRows, dn, err := readUvarint(b, off)
	if err != nil {
		return nil, err
	}
	off += dn
	nStr, dn2, err := readUvarint(b, off)
	if err != nil {
		return nil, err
	}
	off += dn2
	if nStr > maxSliceLen {
		return nil, errCorrupt
	}
	strs := make([]string, 0, int(nStr))
	for i := uint64(0); i < nStr; i++ {
		l, dn3, err := readUvarint(b, off)
		if err != nil {
			return nil, err
		}
		off += dn3
		if l > maxSliceLen || off+int(l) > len(b) {
			return nil, errCorrupt
		}
		strs = append(strs, string(b[off:off+int(l)]))
		off += int(l)
	}
	return &Decoder{strs: strs, buf: b, off: off, nRows: nRows}, nil
}

// NumRows 表行数（文件头记录）。
func (d *Decoder) NumRows() uint64 { return d.nRows }

func (d *Decoder) need(n int) error {
	if d.off+n > len(d.buf) {
		return errCorrupt
	}
	return nil
}

// ReadInt64Zigzag 有符号整数（int64 / 与 int 共用线格式）。
func (d *Decoder) ReadInt64Zigzag() (int64, error) {
	v, n, err := readUvarint(d.buf, d.off)
	if err != nil {
		return 0, err
	}
	d.off += n
	return dezigzag64(v), nil
}

// ReadInt32Zigzag 用于 int32 / 枚举底层。
func (d *Decoder) ReadInt32Zigzag() (int32, error) {
	v, err := d.ReadInt64Zigzag()
	if err != nil {
		return 0, err
	}
	return int32(v), nil
}

// ReadInt 从 zigzag64 解码并转为 int（适用于本机 int）。
func (d *Decoder) ReadInt() (int, error) {
	v, err := d.ReadInt64Zigzag()
	if err != nil {
		return 0, err
	}
	return int(v), nil
}

// ReadFloat64 little-endian IEEE754。
func (d *Decoder) ReadFloat64() (float64, error) {
	if err := d.need(8); err != nil {
		return 0, err
	}
	u := binary.LittleEndian.Uint64(d.buf[d.off : d.off+8])
	d.off += 8
	return math.Float64frombits(u), nil
}

// ReadString 从字符串池按索引取出。
func (d *Decoder) ReadString() (string, error) {
	idx, err := d.readUvarintLocal()
	if err != nil {
		return "", err
	}
	if idx >= uint64(len(d.strs)) {
		return "", errCorrupt
	}
	return d.strs[idx], nil
}

func (d *Decoder) readUvarintLocal() (uint64, error) {
	v, n, err := readUvarint(d.buf, d.off)
	if err != nil {
		return 0, err
	}
	d.off += n
	return v, nil
}

// ReadIntSlice zigzag 子元素。
func (d *Decoder) ReadIntSlice() ([]int, error) {
	n, err := d.readUvarintLocal()
	if err != nil {
		return nil, err
	}
	if n > maxSliceLen {
		return nil, errCorrupt
	}
	out := make([]int, 0, n)
	for i := uint64(0); i < n; i++ {
		v, err := d.ReadInt64Zigzag()
		if err != nil {
			return nil, err
		}
		out = append(out, int(v))
	}
	return out, nil
}

// ReadInt64Slice 每元素 zigzag64。
func (d *Decoder) ReadInt64Slice() ([]int64, error) {
	n, err := d.readUvarintLocal()
	if err != nil {
		return nil, err
	}
	if n > maxSliceLen {
		return nil, errCorrupt
	}
	out := make([]int64, 0, n)
	for i := uint64(0); i < n; i++ {
		v, err := d.ReadInt64Zigzag()
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// ReadFloat64Slice 每元素 8 字节 LE。
func (d *Decoder) ReadFloat64Slice() ([]float64, error) {
	n, err := d.readUvarintLocal()
	if err != nil {
		return nil, err
	}
	if n > maxSliceLen {
		return nil, errCorrupt
	}
	out := make([]float64, 0, n)
	for i := uint64(0); i < n; i++ {
		f, err := d.ReadFloat64()
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, nil
}

// ReadStringSlice 每元素为池索引。
func (d *Decoder) ReadStringSlice() ([]string, error) {
	n, err := d.readUvarintLocal()
	if err != nil {
		return nil, err
	}
	if n > maxSliceLen {
		return nil, errCorrupt
	}
	out := make([]string, 0, n)
	for i := uint64(0); i < n; i++ {
		idx, err := d.readUvarintLocal()
		if err != nil {
			return nil, err
		}
		if idx >= uint64(len(d.strs)) {
			return nil, errCorrupt
		}
		out = append(out, d.strs[idx])
	}
	return out, nil
}

// ReadInt32ZigzagSlice 用于 []枚举。
func (d *Decoder) ReadInt32ZigzagSlice() ([]int32, error) {
	n, err := d.readUvarintLocal()
	if err != nil {
		return nil, err
	}
	if n > maxSliceLen {
		return nil, errCorrupt
	}
	out := make([]int32, 0, n)
	for i := uint64(0); i < n; i++ {
		v, err := d.ReadInt32Zigzag()
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// ReadJSONBytes 读取 uvarint 长度前缀的原始 JSON 字节（仅引用底层缓冲至下一次 Read 调用前有效）。
func (d *Decoder) ReadJSONBytes() ([]byte, error) {
	n, err := d.readUvarintLocal()
	if err != nil {
		return nil, err
	}
	if n > maxSliceLen || d.off+int(n) > len(d.buf) {
		return nil, errCorrupt
	}
	slice := d.buf[d.off : d.off+int(n)]
	d.off += int(n)
	return slice, nil
}

// UnmarshalJSON 读取一段长度前缀 JSON 并 json.Unmarshal 到 v。
func (d *Decoder) UnmarshalJSON(v interface{}) error {
	raw, err := d.ReadJSONBytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, v)
}

// Remaining 返回尚未消费的字节数（用于调试）。
func (d *Decoder) Remaining() int {
	return len(d.buf) - d.off
}

// ErrTrailing 若解析完 nRows 后仍有未读字节则返回错误（可选校验）。
func (d *Decoder) ErrIfTrailing() error {
	if d.off != len(d.buf) {
		return fmt.Errorf("tablebin: %d trailing bytes", len(d.buf)-d.off)
	}
	return nil
}
