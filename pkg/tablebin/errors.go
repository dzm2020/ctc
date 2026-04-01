package tablebin

import "errors"

var (
	errCorrupt = errors.New("tablebin: corrupt or truncated file")
	errVersion = errors.New("tablebin: unsupported format version")
)
