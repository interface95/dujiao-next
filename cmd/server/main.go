package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/dujiao-next/internal/app"
	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/logger"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/version"

	"github.com/gin-gonic/gin"
)

const (
	ansiReset     = "\033[0m"
	ansiBold      = "\033[1m"
	ansiDim       = "\033[2m"
	ansiGreen     = "\033[32m"
	ansiBlue      = "\033[34m"
	ansiCyan      = "\033[36m"
	ansiBrightMag = "\033[95m"
)

func main() {
	printStartupBanner()

	// 加载配置
	cfg := config.Load()
	logger.Init(cfg.Server.Mode, cfg.Log.ToLoggerOptions())
	stdLog := logger.StdLogger()

	if cfg.Server.Mode == "release" {
		if isWeakSecret(cfg.JWT.SecretKey) {
			stdLog.Fatalf("JWT secret 过弱或仍为默认值，请在生产环境中配置强随机密钥")
		}
	} else if isWeakSecret(cfg.JWT.SecretKey) {
		stdLog.Printf("警告: JWT secret 过弱或仍为默认值，建议在生产环境中更换")
	}

	// 初始化数据库
	if err := models.InitDB(cfg.Database.Driver, cfg.Database.DSN, models.DBPoolConfig{
		MaxOpenConns:           cfg.Database.Pool.MaxOpenConns,
		MaxIdleConns:           cfg.Database.Pool.MaxIdleConns,
		ConnMaxLifetimeSeconds: cfg.Database.Pool.ConnMaxLifetimeSeconds,
		ConnMaxIdleTimeSeconds: cfg.Database.Pool.ConnMaxIdleTimeSeconds,
	}); err != nil {
		stdLog.Fatalf("数据库初始化失败: %v", err)
	}

	// 自动迁移数据库表
	if err := models.AutoMigrate(); err != nil {
		stdLog.Fatalf("数据库迁移失败: %v", err)
	}

	// 初始化默认管理员账号
	defaultAdminUser, defaultAdminPass := resolveDefaultAdminCredentials(cfg)
	if cfg.Server.Mode == "release" && defaultAdminPass == "" {
		stdLog.Printf("警告: 未设置 DJ_DEFAULT_ADMIN_PASSWORD 且 bootstrap.default_admin_password 为空，已跳过默认管理员初始化")
	} else if err := models.InitDefaultAdmin(defaultAdminUser, defaultAdminPass); err != nil {
		stdLog.Printf("警告: 初始化默认管理员失败: %v", err)
	}

	// 设置 Gin 模式
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 解析命令行参数
	var mode string
	flag.StringVar(&mode, "mode", app.ModeAll, "启动模式: all (默认), api, worker")
	flag.Parse()

	if err := app.Run(app.Options{
		Config:  cfg,
		Logger:  logger.S(),
		Signals: []os.Signal{syscall.SIGINT, syscall.SIGTERM},
		Mode:    mode,
	}); err != nil {
		stdLog.Fatalf("服务运行失败: %v", err)
	}
}

func printStartupBanner() {
	fmt.Println(ansiBrightMag + "╔══════════════════════════════════════════════════════════════════════╗" + ansiReset)
	fmt.Println(ansiBrightMag + "║                      🚀 Dujiao-Next API 启动中                      ║" + ansiReset)
	fmt.Println(ansiBrightMag + "╚══════════════════════════════════════════════════════════════════════╝" + ansiReset)
	fmt.Println(ansiCyan + "██████╗ ██╗   ██╗     ██╗ █████╗  ██████╗      ███╗   ██╗███████╗██╗  ██╗████████╗" + ansiReset)
	fmt.Println(ansiCyan + "██╔══██╗██║   ██║     ██║██╔══██╗██╔═══██╗     ████╗  ██║██╔════╝╚██╗██╔╝╚══██╔══╝" + ansiReset)
	fmt.Println(ansiCyan + "██║  ██║██║   ██║     ██║███████║██║   ██║     ██╔██╗ ██║█████╗   ╚███╔╝    ██║   " + ansiReset)
	fmt.Println(ansiCyan + "██║  ██║██║   ██║██   ██║██╔══██║██║   ██║     ██║╚██╗██║██╔══╝   ██╔██╗    ██║   " + ansiReset)
	fmt.Println(ansiCyan + "██████╔╝╚██████╔╝╚█████╔╝██║  ██║╚██████╔╝     ██║ ╚████║███████╗██╔╝ ██╗   ██║   " + ansiReset)
	fmt.Println(ansiCyan + "╚═════╝  ╚═════╝  ╚════╝ ╚═╝  ╚═╝ ╚═════╝      ╚═╝  ╚═══╝╚══════╝╚═╝  ╚═╝   ╚═╝   " + ansiReset)
	fmt.Println(ansiGreen + ansiBold + "Open Source Repositories" + ansiReset)
	fmt.Println(ansiBlue + "• Root:    https://github.com/dujiao-next" + ansiReset)
	fmt.Println(ansiBlue + "• API:     https://github.com/dujiao-next/dujiao-next" + ansiReset)
	fmt.Println(ansiBlue + "• User:    https://github.com/dujiao-next/user" + ansiReset)
	fmt.Println(ansiBlue + "• Admin:   https://github.com/dujiao-next/admin" + ansiReset)
	fmt.Println(ansiBlue + "• Official:https://github.com/dujiao-next/document" + ansiReset)
	fmt.Println(ansiGreen + "Version: " + version.Version + ansiReset)
	fmt.Println(ansiDim + "--------------------------------------------------------------" + ansiReset)
}

func isWeakSecret(secret string) bool {
	if len(secret) < 32 {
		return true
	}
	normalized := strings.ToLower(secret)
	if strings.Contains(normalized, "change-me") ||
		strings.Contains(normalized, "change-in-production") ||
		strings.Contains(normalized, "your-secret-key") {
		return true
	}
	return false
}

// resolveDefaultAdminCredentials 解析默认管理员初始化凭据（环境变量优先，其次 config.yml）
func resolveDefaultAdminCredentials(cfg *config.Config) (string, string) {
	user := strings.TrimSpace(os.Getenv("DJ_DEFAULT_ADMIN_USERNAME"))
	pass := strings.TrimSpace(os.Getenv("DJ_DEFAULT_ADMIN_PASSWORD"))
	if cfg == nil {
		return user, pass
	}
	if user == "" {
		user = strings.TrimSpace(cfg.Bootstrap.DefaultAdminUsername)
	}
	if pass == "" {
		pass = strings.TrimSpace(cfg.Bootstrap.DefaultAdminPassword)
	}
	return user, pass
}
