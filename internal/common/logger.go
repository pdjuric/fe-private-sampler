package common

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"sync"
)

// todo close files with wgs!!!

var loggerMap map[string]*logrus.Logger
var loggerMapMutex sync.Mutex

// logs dir must exist
func GetLoggerForFile(prefix string, filename string) *Logger {
	loggerMapMutex.Lock()
	defer loggerMapMutex.Unlock()
	if loggerMap == nil {
		loggerMap = make(map[string]*logrus.Logger)
	}

	dumbLogger, exists := loggerMap[filename]
	if !exists {
		//err := os.Mkdir("	logs", 0755)
		//if err != nil {
		//	fmt.Errorf("error creating log dir: %s", err)
		//	return nil
		//}
		logFile, err := os.Create("logs/" + filename + ".log")
		if err != nil {
			fmt.Errorf("error creating log file: %s", err)
			return nil
		}

		dumbLogger = logrus.New()
		dumbLogger.SetLevel(logrus.DebugLevel)
		dumbLogger.SetOutput(logFile)
		//httpLogger.SetReportCaller(true)
		//httpLogger.SetFormatter(&logFormatter{})
		loggerMap[filename] = dumbLogger
	}
	return &Logger{" [" + prefix + "] ", dumbLogger}
}

func GetLogger(prefix string, logger *Logger) *Logger {
	return &Logger{" [" + prefix + "] ", logger.dumbLogger}
}

func GetDiscardLogger() *Logger {
	dumbLogger := logrus.New()
	dumbLogger.SetOutput(io.Discard)
	return &Logger{"", dumbLogger}
}

type Logger struct {
	prefix     string
	dumbLogger *logrus.Logger
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.dumbLogger.Infof(l.prefix+format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.dumbLogger.Errorf(l.prefix+format, args...)
}

func (l *Logger) Err(err error) {
	l.dumbLogger.Error(l.prefix + err.Error())
}
