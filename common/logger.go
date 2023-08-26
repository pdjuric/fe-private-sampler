package common

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"strings"
	"sync"
)

var loggerMap map[string]*logrus.Logger
var loggerMapMutex sync.Mutex
var loggingDir = ""

// InitLogger initializes the logger ; must be called before any logging
func InitLogger(dir string) error {
	loggerMap = make(map[string]*logrus.Logger)

	//err := os.Mkdir(dir, 0755)
	//if err != nil {
	//	return fmt.Errorf("error during creating log dir %s, using current working dir for logging...", dir)
	//}

	if dir[len(dir)-1] != '/' {
		dir += "/"
	}
	loggingDir = "logs/" + dir
	return nil
	// todo close files
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
		dumbLogger.SetFormatter(&LogFormatter{
			logrus.TextFormatter{
				FullTimestamp:          true,
				TimestampFormat:        "2006-01-02 15:04:05",
				ForceColors:            true,
				DisableLevelTruncation: true,
			},
		})
		loggerMap[filename] = dumbLogger
	}

	prefix = prefix + "$"

	return &Logger{prefix, dumbLogger}
}

func GetLogger(prefix string, logger *Logger) *Logger {
	prefix = prefix + "$"
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

func (l *Logger) Debug(format string, args ...interface{}) {
	l.dumbLogger.Debugf(l.prefix+format, args...)
}

func (l *Logger) Err(err error) {
	l.dumbLogger.Error(l.prefix + err.Error())
}

type LogFormatter struct {
	logrus.TextFormatter
}

func (f *LogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	prefix := strings.Split(entry.Message, "$")[0]
	message, _ := strings.CutPrefix(entry.Message, prefix+"$")
	if len(prefix) > 0 {
		prefix = "[" + prefix + "] "
	}
	return []byte(fmt.Sprintf("[%s] %s %s- %s\n", entry.Time.Format(f.TimestampFormat), strings.ToUpper(entry.Level.String()), prefix, message)), nil
}
