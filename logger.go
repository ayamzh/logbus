package logbus

import (
	"github.com/sandwich-go/logbus/basics"
	"github.com/sandwich-go/logbus/config"
	"github.com/sandwich-go/logbus/fluentd"
	"github.com/sandwich-go/logbus/global"
	"github.com/sandwich-go/logbus/monitor"
	"github.com/sandwich-go/logbus/stdl"
	"go.uber.org/zap"
)

var stdLogger *stdl.StdLogger

func init() {
	Init(config.NewConf())
}

func Init(conf *config.Conf) {
	basics.InitBasic(conf)
	stdLogger = stdl.NewDefaultStdLogger(basics.BasicLogger, []string{DefaultTag})
	if conf.OutputFluentd {
		fluentd.Init(conf.FluentdConfig)
	}
	ImplementGLog(conf.PrintAsError)
	monitor.SetDefaultMetricsReporter(conf.MonitorOutput,
		conf.DefaultPrometheusListenAddress,
		conf.DefaultPrometheusPath,
		conf.DefaultPercentiles,
		conf.DefaultLabel,
		conf.MonitorTimingMaxAge)
	if conf.MonitorOutput == config.Prometheus {
		DebugWithChannel(Monitor, zap.String("prometheus [http] listening on", conf.DefaultPrometheusListenAddress), zap.String("path", conf.DefaultPrometheusPath))
	}
}

func Logger(tags ...string) ILogger {
	return stdl.GetStdLogger(tags...)
}

func Tracker(tags ...string) ITracker {
	return GetStdLogger(tags...)
}

func GetStdLogger(tags ...string) *stdl.StdLogger {
	return stdl.GetStdLogger(tags...)
}

func Close() {
	_ = stdLogger.Sync()
	PurgeLoggerMap()
	fluentd.Close()
}

func PurgeLoggerMap() { //used by logbus
	global.LoggerMap.Range(func(key, value interface{}) bool {
		_ = value.(*stdl.StdLogger).Sync()
		return true
	})
}
