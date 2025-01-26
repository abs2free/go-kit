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
	FilePath string
}

// 默认日志配置
var DefaultConfig = &LoggerConfig{
	Level:    zap.InfoLevel,
	FilePath: "logs/zap.log",
	Rotate: lumberjack.Logger{
		MaxSize:    20,
		MaxAge:     30,
		MaxBackups: 50,
		Compress:   false,
		Filename:   "logs/zap.log",
	},
	Encoder: zapcore.EncoderConfig{
		LevelKey:       "level",
		NameKey:        "logs/zap.log",
		TimeKey:        "time",
		MessageKey:     "msg",
		CallerKey:      "caller",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder, // 默认小写编码器
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	},
}

// CustomLevelEncoder 自定义日志级别编码器
func CustomLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	var coloredLevel string
	switch level {
	case zapcore.DebugLevel:
		coloredLevel = "\x1b[37mDEBUG\x1b[0m" // 白色
	case zapcore.InfoLevel:
		coloredLevel = "\x1b[32mINFO\x1b[0m" // 绿色
	case zapcore.WarnLevel:
		coloredLevel = "\x1b[33mWARN\x1b[0m" // 黄色
	case zapcore.ErrorLevel:
		coloredLevel = "\x1b[31mERROR\x1b[0m" // 红色
	case zapcore.DPanicLevel:
		coloredLevel = "\x1b[35mDPANIC\x1b[0m" // 紫色
	case zapcore.PanicLevel:
		coloredLevel = "\x1b[35mPANIC\x1b[0m" // 紫色
	case zapcore.FatalLevel:
		coloredLevel = "\x1b[35mFATAL\x1b[0m" // 紫色
	default:
		coloredLevel = level.String()
	}
	enc.AppendString(coloredLevel)
}

// CustomTimeEncoder 自定义时间编码器
func CustomTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("\x1b[36m" + t.Format("2006-01-02 15:04:05.000") + "\x1b[0m") // 青色
}

// Option 配置选项
type Option func(*LoggerConfig)

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

type CoreBuilder func(*zapcore.Core)

func WithFileCore(options ...Option) CoreBuilder {
	return func(core *zapcore.Core) {
		cfg := *DefaultConfig
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

func WithConsoleCore(options ...Option) CoreBuilder {
	return func(core *zapcore.Core) {
		cfg := *DefaultConfig

		// 为控制台设置彩色编码器
		cfg.Encoder.EncodeLevel = CustomLevelEncoder
		cfg.Encoder.EncodeTime = CustomTimeEncoder

		for _, opt := range options {
			opt(&cfg)
		}

		*core = zapcore.NewCore(
			zapcore.NewConsoleEncoder(cfg.Encoder),
			zapcore.AddSync(os.Stdout),
			cfg.Level,
		)
	}
}

// 添加颜色控制选项
func WithColorOutput(enabled bool) Option {
	return func(cfg *LoggerConfig) {
		if !enabled {
			cfg.Encoder.EncodeLevel = zapcore.LowercaseLevelEncoder
			cfg.Encoder.EncodeTime = zapcore.ISO8601TimeEncoder
			cfg.Encoder.EncodeCaller = zapcore.ShortCallerEncoder
			cfg.Encoder.EncodeLevel = zapcore.CapitalColorLevelEncoder
		}
	}
}

func new(builders ...CoreBuilder) (*zap.SugaredLogger, error) {
	cores := make([]zapcore.Core, 0, len(builders))

	if len(builders) == 0 {
		return nil, fmt.Errorf("at least one core builder is required")
	}

	for _, builder := range builders {
		if builder == nil {
			continue
		}

		var core zapcore.Core
		builder(&core)

		if core == nil {
			continue
		}

		cores = append(cores, core)
	}

	if len(cores) == 0 {
		return nil, fmt.Errorf("no valid log cores were configured")
	}

	opts := []zap.Option{
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zap.ErrorLevel),
	}

	logger := zap.New(
		zapcore.NewTee(cores...),
		opts...,
	)

	if Logger != nil {
		_ = Logger.Sync()
	}

	Logger = logger.Sugar()

	return Logger, nil
}

func newJSONEncoder(cfg *LoggerConfig) zapcore.Encoder {
	cfg.Encoder.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
	}
	return zapcore.NewJSONEncoder(cfg.Encoder)
}

func newFileWriter(cfg *LoggerConfig) zapcore.WriteSyncer {
	writer := &cfg.Rotate
	return zapcore.AddSync(writer)
}

func New(level zapcore.Level) (*zap.SugaredLogger, error) {
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}

	fileCore := WithFileCore(
		WithRotateSettings(10, 7, true),
		WithLogLevel(level),
	)

	consoleCore := WithConsoleCore(
		WithLogLevel(level),
		WithColorOutput(true), // 或 false 禁用颜色
	)

	logger, err := new(fileCore, consoleCore)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		return nil, err
	}

	logger.Info("Logger initialized with fileCore and consoleCore!")

	return logger, nil
}
