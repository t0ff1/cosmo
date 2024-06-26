package logging

import (
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	requestIDField = "reqId"
)

type RequestIDKey struct{}

func New(prettyLogging bool, debug bool, level zapcore.Level, fileLogging bool, fileName string) *zap.Logger {
	return newZapLogger(zapcore.AddSync(os.Stdout), prettyLogging, debug, level, fileLogging, fileName)
}

func zapBaseEncoderConfig() zapcore.EncoderConfig {
	ec := zap.NewProductionEncoderConfig()
	ec.EncodeDuration = zapcore.SecondsDurationEncoder
	ec.TimeKey = "time"
	return ec
}

func ZapJsonEncoder() zapcore.Encoder {
	ec := zapBaseEncoderConfig()
	ec.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		nanos := t.UnixNano()
		millis := int64(math.Trunc(float64(nanos) / float64(time.Millisecond)))
		enc.AppendInt64(millis)
	}
	return zapcore.NewJSONEncoder(ec)
}

func zapConsoleEncoder() zapcore.Encoder {
	ec := zapBaseEncoderConfig()
	ec.ConsoleSeparator = " "
	ec.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05 PM")
	ec.EncodeLevel = zapcore.CapitalColorLevelEncoder
	return zapcore.NewConsoleEncoder(ec)
}

// WIP: fileName should be a log file prefix and using rotating logs with configurable file size
func zapFileCore(fileName string, logLevel zapcore.Level) (zapcore.Core, error) {
	logFile, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	jsonEncoder := ZapJsonEncoder()
	fileWriter := zapcore.AddSync(logFile)
	fileCore := zapcore.NewCore(jsonEncoder, fileWriter, logLevel)

	return fileCore, nil
}

func attachBaseFields(logger *zap.Logger) *zap.Logger {
	host, err := os.Hostname()
	if err != nil {
		host = "unknown"
	}

	logger = logger.With(
		zap.String("hostname", host),
		zap.Int("pid", os.Getpid()),
	)

	return logger
}

func newZapLogger(syncer zapcore.WriteSyncer, prettyLogging bool, debug bool, level zapcore.Level, fileLogging bool, fileName string) *zap.Logger {
	var encoder zapcore.Encoder
	var zapOpts []zap.Option

	if prettyLogging {
		encoder = zapConsoleEncoder()
	} else {
		encoder = ZapJsonEncoder()
	}

	if debug {
		zapOpts = append(zapOpts, zap.AddCaller())
	}

	zapOpts = append(zapOpts, zap.AddStacktrace(zap.ErrorLevel))

	core := zapcore.NewCore(
		encoder,
		syncer,
		level,
	)

	if fileLogging {
		fileCore, err := zapFileCore(fileName, level)
		if err != nil {
			// standard logging, since we don't have a logger yet
			log.Printf("Can't create file logger with file name %s. Error: %v", fileName, err)
		} else {
			core = zapcore.NewTee(fileCore, core)
		}
	}

	zapLogger := zap.New(core, zapOpts...)

	if prettyLogging {
		return zapLogger
	}

	zapLogger = attachBaseFields(zapLogger)

	return zapLogger
}

func ZapLogLevelFromString(logLevel string) (zapcore.Level, error) {
	switch strings.ToUpper(logLevel) {
	case "DEBUG":
		return zap.DebugLevel, nil
	case "INFO":
		return zap.InfoLevel, nil
	case "WARNING":
		return zap.WarnLevel, nil
	case "ERROR":
		return zap.ErrorLevel, nil
	case "FATAL":
		return zap.FatalLevel, nil
	case "PANIC":
		return zap.PanicLevel, nil
	default:
		return -1, fmt.Errorf("unknown log level: %s", logLevel)
	}
}

func WithRequestID(reqID string) zap.Field {
	return zap.String(requestIDField, reqID)
}
