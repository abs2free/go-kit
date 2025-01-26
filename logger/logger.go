package logger

import (
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Logger *zap.SugaredLogger

// LoggerConfig 日志配置
type LoggerConfig struct {
	Encoder zapcore.EncoderConfig
	Rotate  lumberjack.Logger
	Level   zapcore.Level
}

// 默认日志配置
var DefaultConfig = &LoggerConfig{
	Level: zap.InfoLevel,
	Rotate: lumberjack.Logger{
		MaxSize:    20,
		MaxAge:     30,
		MaxBackups: 50,
		Compress:   false,
	},
	Encoder: zapcore.EncoderConfig{
		LevelKey:       "level",
		NameKey:        "log/zap.log",
		TimeKey:        "time",
		MessageKey:     "msg",
		CallerKey:      "caller",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	},
}

// Option 配置选项
type Option func(*LoggerConfig)

// 配置选项方法
func WithLogLevel(level zapcore.Level) Option {
	return func(cfg *LoggerConfig) {
		cfg.Level = level
	}
}

func WithLogFilePath(filePath string) Option {
	return func(cfg *LoggerConfig) {
		cfg.Encoder.NameKey = filePath
	}
}

func WithLogFormat(format zapcore.EncoderConfig) Option {
	return func(cfg *LoggerConfig) {
		cfg.Encoder = format
	}
}

func WithRotateSettings(maxSize, maxAge int, compress bool) Option {
	return func(cfg *LoggerConfig) {
		cfg.Rotate.MaxSize = maxSize
		cfg.Rotate.MaxAge = maxAge
		cfg.Rotate.Compress = compress
	}
}

// CoreBuilder 用于构建日志核心
type CoreBuilder func(*zapcore.Core)

// 配置文件日志核心
func WithFileCore(options ...Option) CoreBuilder {
	return func(core *zapcore.Core) {
		cfg := *DefaultConfig // 创建配置的副本
		for _, opt := range options {
			opt(&cfg)
		}
		*core = zapcore.NewCore(
			newJSONEncoder(&cfg),
			newFileWriter(&cfg),
			cfg.Level,
		)
	}
}

// 配置控制台日志核心
func WithConsoleCore(options ...Option) CoreBuilder {
	return func(core *zapcore.Core) {
		cfg := *DefaultConfig
		for _, opt := range options {
			opt(&cfg)
		}
		cfg.Encoder.EncodeLevel = zapcore.CapitalColorLevelEncoder
		*core = zapcore.NewCore(
			zapcore.NewConsoleEncoder(cfg.Encoder),
			zapcore.AddSync(os.Stdout),
			cfg.Level,
		)
	}
}

// New 初始化日志
func New(fileCoreBuilder, consoleCoreBuilder CoreBuilder) (*zap.SugaredLogger, error) {
	var cores []zapcore.Core

	if fileCoreBuilder != nil {
		var fileCore zapcore.Core
		fileCoreBuilder(&fileCore)
		cores = append(cores, fileCore)
	}

	if consoleCoreBuilder != nil {
		var consoleCore zapcore.Core
		consoleCoreBuilder(&consoleCore)
		cores = append(cores, consoleCore)
	}

	if len(cores) == 0 {
		return nil, fmt.Errorf("no log cores configured")
	}

	core := zapcore.NewTee(cores...)
	logger := zap.New(core, zap.AddCaller())
	Logger = logger.Sugar()
	return Logger, nil
}

// 构造 JSON 格式日志编码器
func newJSONEncoder(cfg *LoggerConfig) zapcore.Encoder {
	cfg.Encoder.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
	}
	return zapcore.NewJSONEncoder(cfg.Encoder)
}

// 构造文件日志写入器
func newFileWriter(cfg *LoggerConfig) zapcore.WriteSyncer {
	writer := &cfg.Rotate
	return zapcore.AddSync(writer)
}

func main() {
	// 配置 fileCore
	fileCore := WithFileCore(
		WithLogFilePath("test_logs/test.log"),
		WithRotateSettings(10, 7, true),
		WithLogLevel(zap.InfoLevel),
	)

	// 配置 consoleCore
	consoleCore := WithConsoleCore(
		WithLogLevel(zap.DebugLevel),
	)

	// 初始化日志
	logger, err := New(fileCore, consoleCore)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		return
	}

	logger.Info("Logger initialized with fileCore and consoleCore!")

}
