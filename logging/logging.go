package logging

import (
	"fmt"
	"os"
	"strings"
	"sync"

	tmlog "github.com/cometbft/cometbft/libs/log"
	log "github.com/xlab/suplog"
	bugsnagHook "github.com/xlab/suplog/hooks/bugsnag"
	debugHook "github.com/xlab/suplog/hooks/debug"
)

func Level(s string) log.Level {
	switch s {
	case "1", "error":
		return log.ErrorLevel
	case "2", "warn":
		return log.WarnLevel
	case "3", "info":
		return log.InfoLevel
	case "4", "debug":
		return log.DebugLevel
	default:
		return log.FatalLevel
	}
}

func NewWrappedSuplog(minLevel, levelsMap string, useJSON bool) tmlog.Logger {
	var defaultLevel log.Level

	parsedLevelsMap := parseLevelsMap(levelsMap)
	if ll, ok := parsedLevelsMap["*"]; ok {
		// wildcard is defined, i.e. *:error
		defaultLevel = ll
	} else {
		// no wildcard level
		parsedLevelsMap["*"] = Level(minLevel)
	}

	l := &tmlogWrapper{
		defaultLevel: defaultLevel,
		appLogger:    NewSuplog(Level(minLevel), useJSON),
		levelsMap:    new(sync.Map),
	}

	for k, v := range parsedLevelsMap {
		l.levelsMap.Store(k, v)
	}

	return l
}

func NewSuplog(minLevel log.Level, useJSON bool) log.Logger {
	var formatter log.Formatter

	if useJSON {
		formatter = new(log.JSONFormatter)
	} else {
		formatter = new(log.TextFormatter)
	}

	appLogger := log.NewLogger(os.Stderr, formatter,
		debugHook.NewHook(log.DefaultLogger, &debugHook.HookOptions{
			StackTraceOffset: 1,
		}),
		bugsnagHook.NewHook(log.DefaultLogger, &bugsnagHook.HookOptions{
			Levels: []log.Level{
				log.PanicLevel,
				log.FatalLevel,
				log.ErrorLevel,
				// do not report anything below Error
			},
			StackTraceOffset: 1,
		}),
	)

	appLogger.(log.LoggerConfigurator).SetLevel(minLevel)
	appLogger.(log.LoggerConfigurator).SetStackTraceOffset(1)

	return appLogger
}

type tmlogWrapper struct {
	module       string
	defaultLevel log.Level
	appLogger    log.Logger
	levelsMap    *sync.Map
}

func (l *tmlogWrapper) LevelEnabled(fields log.Fields, currentLevel log.Level) bool {
	var moduleName string
	var logLevelForModule log.Level
	var logLevelSet bool

	if len(l.module) > 0 {
		moduleName = l.module
	} else {
		for k, v := range fields {
			if k == "module" {
				moduleName = v.(string)
				break
			}
		}
	}

	// set log level for module
	ll, ok := l.levelsMap.Load(moduleName)
	if ok { // no level for this module
		logLevelSet = true
	} else {
		ll, logLevelSet = l.levelsMap.Load("*")
	}

	if logLevelSet {
		logLevelForModule = ll.(log.Level)
	} else {
		// defaults
		logLevelForModule = l.defaultLevel
	}

	// Error (1) < Debug (4)
	if currentLevel <= logLevelForModule {
		return true
	}

	return false
}

func (l *tmlogWrapper) Debug(msg string, keyvals ...interface{}) {
	fields := kvToFields(keyvals...)

	if !l.LevelEnabled(fields, log.DebugLevel) {
		return
	}

	if fields != nil {
		l.appLogger.WithFields(fields).Debugln(msg)
		return
	}

	l.appLogger.Debugln(msg)
}

func (l *tmlogWrapper) Info(msg string, keyvals ...interface{}) {
	fields := kvToFields(keyvals...)

	if !l.LevelEnabled(fields, log.InfoLevel) {
		return
	}

	if fields != nil {
		l.appLogger.WithFields(fields).Infoln(msg)
		return
	}

	l.appLogger.Infoln(msg)
}

func (l *tmlogWrapper) Error(msg string, keyvals ...interface{}) {
	fields := kvToFields(keyvals...)

	if !l.LevelEnabled(fields, log.ErrorLevel) {
		return
	}

	if fields != nil {
		l.appLogger.WithFields(fields).Errorln(msg)
		return
	}

	l.appLogger.Errorln(msg)
}

func (l *tmlogWrapper) With(keyvals ...interface{}) tmlog.Logger {
	fields := kvToFields(keyvals...)
	if fields == nil {
		return l
	}

	wrapper := &tmlogWrapper{
		module:       l.module,
		defaultLevel: l.defaultLevel,
		levelsMap:    l.levelsMap,
		appLogger:    l.appLogger.WithFields(fields),
	}

	// check if this With() sets the module name
	for k, v := range fields {
		if k == "module" {
			wrapper.module = v.(string)
			return wrapper
		}
	}

	return wrapper
}

func kvToFields(keyvals ...interface{}) log.Fields {
	if kvLen := len(keyvals); kvLen == 0 || kvLen%2 != 0 {
		return nil
	}

	fields := make(log.Fields)
	for i := 0; i < len(keyvals); i += 2 {
		key, ok := keyvals[i].(string)
		if !ok {
			log.WithField("arg", keyvals[i]).Errorf("argument must be a string, got %T", keyvals[i])
			return nil
		}

		if str, isStringer := keyvals[i+1].(stringer); isStringer {
			fields[key] = str.String()
		} else {
			fields[key] = fmt.Sprintf("%+v", keyvals[i+1])
		}
	}

	return fields
}

type stringer interface {
	String() string
}

// parseLevelsMap is an in-house implementation of parser
// for log levels spec per module, i.e. "main:info,staking:info,*:error"
func parseLevelsMap(levelsMap string) map[string]log.Level {
	list := strings.Split(levelsMap, ",")
	if len(list) == 1 {
		parts := strings.Split(list[0], ":")
		if len(parts) == 1 {
			// only global level set, i.e. "info"
			return map[string]log.Level{
				"*": Level(strings.TrimSpace(parts[0])),
			}
		} else if len(parts) != 2 {
			panic(fmt.Sprintf("wrong level spec: %s", list[0]))
		}

		// "staking:error"
		return map[string]log.Level{
			strings.TrimSpace(parts[0]): Level(strings.TrimSpace(parts[1])),
		}
	}

	levels := make(map[string]log.Level, len(list))
	// ok, just parse levels per module
	for idx, item := range list {
		parts := strings.Split(item, ":")
		if len(parts) != 2 {
			panic(fmt.Sprintf("wrong log level spec: %s (item %d in %s)", item, idx, levelsMap))
		}

		levels[strings.TrimSpace(parts[0])] = Level(strings.TrimSpace(parts[1]))
	}

	return levels
}
