package logger

import (
	"fmt"
	"os"
	"testing"

	"go.uber.org/zap"
)

// 测试日志初始化时 fileCore 和 consoleCore 同时存在的场景
func TestLoggerWithFileAndConsoleCore(t *testing.T) {
	fileCore := WithFileCore(
		WithLogFilePath("test_logs/test.log"),
		WithRotateSettings(5, 3, true),
		WithLogLevel(zap.InfoLevel),
	)

	consoleCore := WithConsoleCore(
		WithLogLevel(zap.DebugLevel),
	)

	logger, err := new(fileCore, consoleCore)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer cleanUpLogFiles()

	logger.Info("This is an INFO message for fileCore and consoleCore.")
	logger.Debug("This is a DEBUG message for fileCore and consoleCore.") // Only visible in consoleCore
	logger.Warn("This is a WARN message for both cores.")
}

// 测试日志初始化时仅 fileCore 存在的场景
func TestLoggerWithFileCoreOnly(t *testing.T) {
	fileCore := WithFileCore(
		WithLogFilePath("test_logs/file_only.log"),
		WithRotateSettings(10, 7, false),
		WithLogLevel(zap.WarnLevel),
	)

	logger, err := new(fileCore, nil)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer cleanUpLogFiles()

	logger.Info("This INFO message should not appear in fileCore.")
	logger.Warn("This WARN message should appear in fileCore.")
	logger.Error("This ERROR message should appear in fileCore.")
}

// 测试日志初始化时仅 consoleCore 存在的场景
func TestLoggerWithConsoleCoreOnly(t *testing.T) {
	consoleCore := WithConsoleCore(
		WithLogLevel(zap.DebugLevel),
	)

	logger, err := new(nil, consoleCore)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	logger.Debug("This DEBUG message should appear in consoleCore.")
	logger.Info("This INFO message should appear in consoleCore.")
	logger.Warn("This WARN message should appear in consoleCore.")
}

// 测试无任何核心配置的场景（应返回错误）
func TestLoggerWithNoCore(t *testing.T) {
	logger, err := new(nil, nil)
	if err == nil {
		t.Fatalf("Expected error when initializing logger with no cores, got nil")
	}

	if logger != nil {
		t.Fatalf("Expected nil logger when no cores are provided, got: %v", logger)
	}
}

// 清理生成的测试日志文件
func cleanUpLogFiles() {
	err := os.RemoveAll("test_logs")
	if err != nil {
		fmt.Printf("Failed to clean up test logs: %v\n", err)
	}
}
