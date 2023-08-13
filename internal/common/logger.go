package common

import (
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"sync"
)

// todo close files with wgs!!!

var loggerMap map[string]*logrus.Logger
var loggerMapMutex sync.Mutex
var loggingDir = ""

// MUST BE CALLED BEFORE ANY LOGGING
func InitLogger(dir string) error {
	loggerMap = make(map[string]*logrus.Logger)

	//err := os.Mkdir(dir, 0755)
	//if err != nil {
	//	return fmt.Errorf("error during creating log dir %s, using current working dir for logging...", dir)
	//}

	if dir[len(dir)-1] != '/' {
		dir += "/"
	}
	loggingDir = dir
	return nil
}

// logs dir must exist
func GetLoggerForFile(prefix string, filename string) *Logger {
	loggerMapMutex.Lock()
	defer loggerMapMutex.Unlock()
	if loggerMap == nil {
		loggerMap = make(map[string]*logrus.Logger)
	}

	dumbLogger, exists := loggerMap[filename]
	if !exists {
		logFile, err := os.Create(loggingDir + filename + ".log")
		if err != nil {
			panic("error during creating log file")
		}

		dumbLogger = logrus.New()
		dumbLogger.SetLevel(logrus.DebugLevel)
		dumbLogger.SetOutput(logFile)
		//httpLogger.SetReportCaller(true)
		//httpLogger.SetFormatter(&logFormatter{})
		loggerMap[filename] = dumbLogger
	}

	if prefix != "" {
		prefix = " [" + prefix + "] "
	}

	return &Logger{prefix, dumbLogger}
}

func GetLogger(prefix string, logger *Logger) *Logger {
	if prefix != "" {
		prefix = " [" + prefix + "] "
	}
	return &Logger{prefix, logger.dumbLogger}
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
