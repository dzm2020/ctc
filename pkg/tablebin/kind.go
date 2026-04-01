package tablebin

// IDKind 主键 id 在二进制中的编码方式（与 @Type「主键」一致）。
type IDKind uint8

const (
	IDInt64 IDKind = iota
	IDInt
	IDString
)

// ColumnKind 表体列的线格式（与生成代码 Read 顺序一致）。
type ColumnKind uint8

const (
	KindInt ColumnKind = iota
	KindInt64
	KindFloat64
	KindString
	KindEnumInt32
	KindSliceInt
	KindSliceInt64
	KindSliceFloat64
	KindSliceString
	KindSliceEnumInt32
	KindStructJSON
	KindSliceStructJSON
)

// Column 描述一列在 map 中的 JSON 键名及其线类型。
type Column struct {
	Key  string
	Kind ColumnKind
}
