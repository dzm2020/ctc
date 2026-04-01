package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	configPath := flag.String("config", "", "配置文件 JSON 路径（必选）")
	flag.Parse()

	if strings.TrimSpace(*configPath) == "" {
		fmt.Fprintln(os.Stderr, "用法: excel2json -config <配置文件.json>")
		os.Exit(1)
	}

	if err := runPipeline(*configPath); err != nil {
		fmt.Fprintf(os.Stderr, "[excel2json] 错误: %v\n", err)
		os.Exit(1)
	}
}
