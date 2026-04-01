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
	// KindSliceStruct 列：SubPath 下为 []map（与 excel 解析一致）；每元素按 SliceElem 顺序写标量/嵌套切片。
	KindSliceStruct
)

// SliceElemField 描述 KindSliceStruct 每个数组元素内的字段顺序（SubPath 为元素 map 内点分路径）。
type SliceElemField struct {
	SubPath string
	Kind    ColumnKind
}

// Column 描述一列在 map 中的 JSON 键名及其线类型。
// SubPath 非空时，值取自 row[Key] 经点路径进入嵌套 map（与 JSON 对象结构一致）；空表示值即 row[Key]。
type Column struct {
	Key       string
	Kind      ColumnKind
	SubPath   string
	SliceElem []SliceElemField // 仅 KindSliceStruct
}
