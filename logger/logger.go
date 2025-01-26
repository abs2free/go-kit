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
	Encoder  zapcore.EncoderConfig
	Rotate   lumberjack.Logger
	Level    zapcore.Level
	FilePath string // 添加文件路径配置
}

// 默认日志配置
var DefaultConfig = &LoggerConfig{
	Level:    zap.InfoLevel,
	FilePath: "logs/zap.log", // 设置默认日志路径
	Rotate: lumberjack.Logger{
		MaxSize:    20,
		MaxAge:     30,
		MaxBackups: 50,
		Compress:   false,
		Filename:   "logs/zap.log", // 设置日志文件路径
	},
	Encoder: zapcore.EncoderConfig{
		LevelKey:       "level",
		NameKey:        "logs/zap.log",
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
		cfg.FilePath = filePath
		cfg.Rotate.Filename = filePath
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

		cfg.Rotate.Filename = cfg.Encoder.NameKey

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

func new(builders ...CoreBuilder) (*zap.SugaredLogger, error) {
	// 预分配切片容量
	cores := make([]zapcore.Core, 0, len(builders))

	// 验证是否提供了构建器
	if len(builders) == 0 {
		return nil, fmt.Errorf("at least one core builder is required")
	}

	// 构建所有cores
	for _, builder := range builders {
		if builder == nil {
			continue
		}

		var core zapcore.Core
		builder(&core)

		// 验证core是否正确初始化
		if core == nil {
			continue
		}

		cores = append(cores, core)
	}

	// 验证是否有有效的cores
	if len(cores) == 0 {
		return nil, fmt.Errorf("no valid log cores were configured")
	}

	// 配置选项
	opts := []zap.Option{
		zap.AddCaller(),
		zap.AddCallerSkip(1),              // 跳过包装函数调用
		zap.AddStacktrace(zap.ErrorLevel), // 错误时添加堆栈跟踪
	}

	// 创建logger
	logger := zap.New(
		zapcore.NewTee(cores...),
		opts...,
	)

	// 确保之前的logger被正确清理
	if Logger != nil {
		_ = Logger.Sync()
	}

	// 设置全局logger
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

func New(level zapcore.Level) (*zap.SugaredLogger, error) {
	// 确保日志目录存在
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}

	// 配置 fileCore
	fileCore := WithFileCore(
		WithRotateSettings(10, 7, true),
		WithLogLevel(level),
	)

	// 配置 consoleCore
	consoleCore := WithConsoleCore(
		WithLogLevel(level),
	)

	// 初始化日志
	logger, err := new(fileCore, consoleCore)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		return nil, err
	}

	logger.Info("Logger initialized with fileCore and consoleCore!")

	return logger, nil
}
