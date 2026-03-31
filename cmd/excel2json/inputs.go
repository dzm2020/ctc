package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"ctc/internal/config"
)

// resolveXlsxInputs 根据配置 inputs 解析所有 .xlsx（路径可为文件或目录）。
func resolveXlsxInputs(cfg *config.Config) ([]string, error) {
	if cfg == nil || len(cfg.Inputs) == 0 {
		return nil, fmt.Errorf("请在配置中设置 inputs（至少一条目录或 .xlsx 路径）")
	}
	var all []string
	seen := make(map[string]bool)
	for _, raw := range cfg.Inputs {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		xs, err := expandPathToXlsx(raw)
		if err != nil {
			return nil, fmt.Errorf("配置 inputs 项 %q: %w", raw, err)
		}
		for _, x := range xs {
			abs, err := filepath.Abs(x)
			if err != nil {
				abs = x
			}
			if !seen[abs] {
				seen[abs] = true
				all = append(all, x)
			}
		}
	}
	if len(all) == 0 {
		return nil, fmt.Errorf("配置 inputs 未解析到任何 .xlsx")
	}
	sort.Strings(all)
	return all, nil
}

func expandPathToXlsx(path string) ([]string, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		if strings.EqualFold(filepath.Ext(path), ".xlsx") {
			return []string{path}, nil
		}
		return nil, fmt.Errorf("不是 .xlsx 文件: %s", path)
	}
	var out []string
	err = filepath.WalkDir(path, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(p), ".xlsx") {
			out = append(out, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("目录下无 .xlsx: %s", path)
	}
	sort.Strings(out)
	return out, nil
}
