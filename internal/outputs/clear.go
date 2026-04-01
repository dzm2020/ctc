package outputs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ClearDirectory 删除 dir 下所有子项（不删除 dir 本身）。若 dir 不存在则创建空目录。
// 用于生成前清空输出目录，避免残留旧表或旧代码。
func ClearDirectory(dir string) error {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return fmt.Errorf("输出目录路径为空")
	}
	dir = filepath.Clean(dir)
	if dir == "." {
		return fmt.Errorf("拒绝清空不安全的相对路径 %q", dir)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("输出目录 %q: %w", dir, err)
	}
	if isRootDir(abs) {
		return fmt.Errorf("拒绝清空根目录或盘符根 %q", abs)
	}

	fi, err := os.Stat(abs)
	if os.IsNotExist(err) {
		return os.MkdirAll(abs, 0o755)
	}
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return fmt.Errorf("输出路径不是目录: %s", abs)
	}

	entries, err := os.ReadDir(abs)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(abs, e.Name())); err != nil {
			return fmt.Errorf("删除 %s: %w", filepath.Join(abs, e.Name()), err)
		}
	}
	return nil
}

// ClearDirectoriesUnique 对多个目录逐个 ClearDirectory；路径经 Abs 去重，同一目录只清空一次。
func ClearDirectoriesUnique(dirs []string) error {
	seen := make(map[string]struct{})
	for _, d := range dirs {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		abs, err := filepath.Abs(filepath.Clean(d))
		if err != nil {
			return err
		}
		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}
		if err := ClearDirectory(abs); err != nil {
			return fmt.Errorf("%s: %w", abs, err)
		}
	}
	return nil
}

func isRootDir(abs string) bool {
	c := filepath.Clean(abs)
	if c == string(os.PathSeparator) {
		return true
	}
	v := filepath.VolumeName(c)
	if v == "" {
		return false
	}
	rest := strings.TrimPrefix(strings.TrimPrefix(c, v), string(filepath.Separator))
	return rest == ""
}
