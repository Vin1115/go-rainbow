package core

//finished
import (
	"fmt"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"os"
	"time"
)

//初始化引导日志模块
func (r *Rainbow) bootLog() { //todo zapcore用法
	encoder := getEncoder()

	var cores []zapcore.Core

	writeSyncer := getLogWriter(r.cfg.RuntimePath)
	fileCore := zapcore.NewCore(encoder, writeSyncer, zapcore.DebugLevel)
	cores = append(cores, fileCore)

	if r.cfg.Service.Debug {
		consoleDebug := zapcore.Lock(os.Stdout)
		consoleCore := zapcore.NewCore(encoder, consoleDebug, zapcore.DebugLevel)
		cores = append(cores, consoleCore)
	}

	core := zapcore.NewTee(cores...)
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	l := logger.Sugar()
	r.setSafe("log", l) //添加日志模块实例进容器container中

	r.logBoot = 1 //设置初始化标志，在Log函数中判断是否使用标准库log输出
}

//格式化日志记录字符串
func logFormat(label string, log interface{}) string {
	e := fmt.Sprintf("[%s] %s", label, log)
	return e
}

//按照格式地去写进日志文件，如果debug设置了true则同时打印到shell界面
func (r *Rainbow) Log(level logLevel, label string, data interface{}) {
	l := r.GetLog() //取容器container内的日志实例
	format := logFormat(label, data)
	switch level {
	case DebugLevel:
		l.Debug(format)
	case InfoLevel:
		l.Info(format)
	case WarnLevel:
		l.Warn(format)
	case ErrorLevel:
		l.Errorf(format)
	case DPanicLevel:
		l.DPanic(format)
	case PanicLevel:
		l.Panic(format)
	case FatalLevel:
		if r.logBoot == 1 { //若无初始化引导日志模块，则用标准库log输出错误
			l.Fatal(format)
		} else {
			log.Fatal(format)
		}
	}
}

//log format encoder config
func getEncoder() zapcore.Encoder {
	encoderConfig := zapcore.EncoderConfig{
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("2006-01-02 15:04:05"))
		},
		TimeKey:      "time",
		LevelKey:     "level",
		NameKey:      "logger",
		CallerKey:    "caller",
		MessageKey:   "msg",
		EncodeLevel:  zapcore.LowercaseLevelEncoder,
		EncodeCaller: zapcore.ShortCallerEncoder,
	}
	return zapcore.NewConsoleEncoder(encoderConfig)
}

//todo 搞清楚这两个包用法
func getLogWriter(runtimePath string) zapcore.WriteSyncer {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   runtimePath + "/logs/log.log",
		MaxSize:    2,
		MaxBackups: 10000,
		MaxAge:     180,
		Compress:   false,
	}
	return zapcore.AddSync(lumberJackLogger)
}
