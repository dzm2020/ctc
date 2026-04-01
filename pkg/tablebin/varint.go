package tablebin

// uvarint 编码同 protobuf：每字节低 7 位有效，最高位表示后续还有字节。

func appendUvarint(b []byte, x uint64) []byte {
	for x >= 0x80 {
		b = append(b, byte(x)|0x80)
		x >>= 7
	}
	return append(b, byte(x))
}

func readUvarint(buf []byte, off int) (v uint64, n int, err error) {
	var shift uint
	for n < len(buf)-off {
		c := buf[off+n]
		n++
		v |= uint64(c&0x7f) << shift
		if c < 0x80 {
			return v, n, nil
		}
		shift += 7
		if shift > 63 {
			return 0, n, errCorrupt
		}
	}
	return 0, n, errCorrupt
}

func zigzag64(i int64) uint64 {
	return uint64((i << 1) ^ (i >> 63))
}

func dezigzag64(u uint64) int64 {
	return int64(u>>1) ^ -(int64(u & 1))
}

func appendZigzag64(b []byte, i int64) []byte {
	return appendUvarint(b, zigzag64(i))
}
