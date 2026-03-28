package main

import (
	"fmt"
	"log"
	"sysmonitord/internal/config"
)

func main() {
	configPath := "./config.yaml"

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("加载配置失败： %v", err)
	}

	fmt.Println("加载配置成功：")
	fmt.Printf("审计配置： %+v\n", cfg.Audit)
	fmt.Printf("扫描配置： %+v\n", cfg.Scanner)

	fmt.Printf("审计服务器地址：%s:%d\n", cfg.Audit.Server, cfg.Audit.Port)
}
