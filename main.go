package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// 命令行参数
	envFile := flag.String("env-file", "", "环境变量文件路径（默认同目录下 .env）")
	dataFlag := flag.String("data", "", "targets.json 路径（默认同目录下的 targets.json）")
	userFlag := flag.String("user", "", "登录用户名（不设则不启用页面认证）")
	passFlag := flag.String("pass", "", "登录密码")
	categrafTokenFlag := flag.String("categraf-token", "", "categraf http_provider 拉取配置用的独立 token（不设则该端点公开，建议设）")
	flag.Parse()

	// 加载 .env 文件
	envPath := *envFile
	if envPath == "" {
		envPath = filepath.Join(execDir(), ".env")
	}
	loadEnv(envPath)

	// 认证：CLI 参数 > 环境变量
	user := firstNonEmpty(*userFlag, os.Getenv("CONFIG_USER"))
	pass := firstNonEmpty(*passFlag, os.Getenv("CONFIG_PASS"))
	InitAuth(user, pass)

	// categraf 拉取 token：与登录账密解耦的独立凭据，仅用于 http_provider 拉配置
	categrafTok := firstNonEmpty(*categrafTokenFlag, os.Getenv("CATEGRAF_TOKEN"))
	InitCategrafToken(categrafTok)

	// 默认配置
	host := getEnv("HOST", "0.0.0.0")
	port := 5000

	dataFile := *dataFlag
	if dataFile == "" {
		dataFile = getEnv("DATA_FILE", filepath.Join(execDir(), "targets.json"))
	}

	// 初始化数据存储
	store, err := NewStore(dataFile)
	if err != nil {
		log.Fatalf("F! init store: %v", err)
	}
	log.Printf("I! data file: %s (%d targets loaded)", dataFile, len(store.All()))

	// 设置路由
	handler := newRouter(store, port)

	// 启动 HTTP 服务
	addr := serverAddr(host, port)
	fmt.Printf("\n")
	fmt.Printf("  ╔══════════════════════════════════════════════════╗\n")
	fmt.Printf("  ║       Categraf 拨测配置管理服务                    ║\n")
	fmt.Printf("  ╠══════════════════════════════════════════════════╣\n")
	fmt.Printf("  ║  Web 管理页面:  http://%s              ║\n", addr)
	fmt.Printf("  ║  Categraf 配置: http://%s/api/config/http_response  ║\n", addr)
	fmt.Printf("  ║  健康检查:     http://%s/health                ║\n", addr)
	fmt.Printf("  ║  数据文件:     %-39s║\n", dataFile)
	fmt.Printf("  ╚══════════════════════════════════════════════════╝\n")
	fmt.Printf("\n")
	fmt.Printf("  部署到服务器后，在 categraf conf/config.toml 中：\n")
	fmt.Printf("    1. providers = [\"local\", \"http\"]\n")
	fmt.Printf("    2. 添加 [http_provider]\n")
	fmt.Printf("       remote_url = \"http://你的IP:%d/api/config/http_response\"\n", port)
	fmt.Printf("       timeout = 5\n")
	fmt.Printf("       reload_interval = 60\n")
	fmt.Printf("\n")
	if CategrafTokenEnabled() {
		fmt.Printf("  categraf 拉取认证: 已启用（Authorization: Bearer <CATEGRAF_TOKEN>）\n")
		fmt.Printf("  categraf config.toml 需配: headers = [\"Authorization\", \"Bearer <token>\"]\n")
	} else {
		fmt.Printf("  ⚠️  /api/config/http_response 未启用认证，建议在 .env 设 CATEGRAF_TOKEN\n")
	}
	fmt.Printf("\n")

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("F! server: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func execDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

// loadEnv 读取 .env 文件中的 KEY=VALUE 到环境变量
func loadEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return // 文件不存在就跳过
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		// TrimLeft 去 UTF-8 BOM（U+FEFF）：PowerShell 的 Set-Content -Encoding UTF8
		// 会在文件头写 BOM，TrimSpace 不认它，会让第一行 key 变成 U+FEFF+CONFIG_USER，
		// 导致 os.Getenv("CONFIG_USER") 查不到、页面认证静默失效。
		// 用 rune(0xFEFF) 而非裸字面量，避免源码里出现非法 BOM 字节。
		line := strings.TrimSpace(strings.TrimLeft(scanner.Text(), string(rune(0xFEFF))))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		// 只在环境变量未设置时注入
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}

// firstNonEmpty 返回第一个非空字符串
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
