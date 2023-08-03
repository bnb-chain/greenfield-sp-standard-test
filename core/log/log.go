package log

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	Blue   = "0;34"
	Red    = "0;31"
	Green  = "0;32"
	Yellow = "0;33"
	Cyan   = "0;36"
	Pink   = "1;35"
)

func Color(code, msg string) string {
	return fmt.Sprintf("\033[%sm%s\033[m", code, msg)
}

const (
	TraceLog = iota
	DebugLog
	InfoLog
	WarnLog
	ErrorLog
	FatalLog
	MaxLevelLog
)

var (
	levels = map[int]string{
		DebugLog: Color(Green, "[DEBUG]"),
		InfoLog:  Color(Cyan, "[INFO]"),
		WarnLog:  Color(Yellow, "[WARN]"),
		ErrorLog: Color(Red, "[ERROR]"),
		FatalLog: Color(Red, "[FATAL]"),
		TraceLog: Color(Pink, "[TRACE]"),
	}
	Stdout = os.Stdout
)

const (
	NAME_PREFIX          = "LEVEL"
	CALL_DEPTH           = 2
	DEFAULT_MAX_LOG_SIZE = 20
	BYTE_TO_MB           = 1024 * 1024
	PATH                 = "./Log/"
)

func GetGID() uint64 {
	var buf [64]byte
	b := buf[:runtime.Stack(buf[:], false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

var Log *Logger

func init() {
	//Default print to console
	InitLog(InfoLog, Stdout)
}
func LevelName(level int) string {
	if name, ok := levels[level]; ok {
		return name
	}
	return NAME_PREFIX + strconv.Itoa(level)
}
func NameLevel(name string) int {
	for k, v := range levels {
		if v == name {
			return k
		}
	}
	var level int
	if strings.HasPrefix(name, NAME_PREFIX) {
		level, _ = strconv.Atoi(name[len(NAME_PREFIX):])
	}
	return level
}

type Logger struct {
	level   int
	logger  *log.Logger
	logFile *os.File
}

func New(out io.Writer, prefix string, flag, level int, file *os.File) *Logger {
	return &Logger{
		level:   level,
		logger:  log.New(out, prefix, flag),
		logFile: file,
	}
}
func (l *Logger) SetDebugLevel(level int) error {
	if level > MaxLevelLog || level < 0 {
		return errors.New("Invalid Debug Level")
	}
	l.level = level
	return nil
}
func (l *Logger) Output(level int, a ...interface{}) error {
	if level >= l.level {
		gid := GetGID()
		gidStr := strconv.FormatUint(gid, 10)
		_, file, line, _ := runtime.Caller(3)
		files := strings.Split(file, "/")
		file = files[len(files)-1]

		a = append([]interface{}{LevelName(level), "GID",
			gidStr, fmt.Sprintf("%s:%d", file, line), ","}, a...)
		return l.logger.Output(CALL_DEPTH, fmt.Sprintln(a...))
	}
	return nil
}
func (l *Logger) Outputf(level int, format string, v ...interface{}) error {
	if level >= l.level {
		gid := GetGID()
		_, file, line, _ := runtime.Caller(3)
		files := strings.Split(file, "/")
		file = files[len(files)-1]
		v = append([]interface{}{LevelName(level), "GID",
			gid, file, line}, v...)

		return l.logger.Output(CALL_DEPTH, fmt.Sprintf("%s %s %d, %s:%d "+format+"\n", v...))
	}
	return nil
}
func (l *Logger) Trace(a ...interface{}) {
	_ = l.Output(TraceLog, a...)
}
func (l *Logger) Tracef(format string, a ...interface{}) {
	_ = l.Outputf(TraceLog, format, a...)
}
func (l *Logger) Debug(a ...interface{}) {
	_ = l.Output(DebugLog, a...)
}
func (l *Logger) Debugf(format string, a ...interface{}) {
	_ = l.Outputf(DebugLog, format, a...)
}
func (l *Logger) Info(a ...interface{}) {
	_ = l.Output(InfoLog, a...)
}
func (l *Logger) Printf(format string, a ...interface{}) {
	_ = l.Outputf(InfoLog, format, a...)
}
func (l *Logger) Infof(format string, a ...interface{}) {
	_ = l.Outputf(InfoLog, format, a...)
}
func (l *Logger) Warn(a ...interface{}) {
	_ = l.Output(WarnLog, a...)
}
func (l *Logger) Warnf(format string, a ...interface{}) {
	_ = l.Outputf(WarnLog, format, a...)
}
func (l *Logger) Error(a ...interface{}) {
	_ = l.Output(ErrorLog, a...)
}
func (l *Logger) Errorf(format string, a ...interface{}) {
	_ = l.Outputf(ErrorLog, format, a...)
}
func (l *Logger) Fatal(a ...interface{}) {
	_ = l.Output(FatalLog, a...)
}
func (l *Logger) Fatalf(format string, a ...interface{}) {
	_ = l.Outputf(FatalLog, format, a...)
}
func Trace(a ...interface{}) {
	if TraceLog < Log.level {
		return
	}
	pc := make([]uintptr, 10)
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	file, line := f.FileLine(pc[0])
	fileName := filepath.Base(file)
	nameFull := f.Name()
	nameEnd := filepath.Ext(nameFull)
	funcName := strings.TrimPrefix(nameEnd, ".")
	a = append([]interface{}{funcName + "()", fileName + ":" + strconv.Itoa(line)}, a...)
	Log.Trace(a...)
}
func Tracef(format string, a ...interface{}) {
	if TraceLog < Log.level {
		return
	}
	pc := make([]uintptr, 10)
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	file, line := f.FileLine(pc[0])
	fileName := filepath.Base(file)
	nameFull := f.Name()
	nameEnd := filepath.Ext(nameFull)
	funcName := strings.TrimPrefix(nameEnd, ".")
	a = append([]interface{}{funcName, fileName, line}, a...)
	Log.Tracef("%s() %s:%d "+format, a...)
}
func Debug(a ...interface{}) {
	if DebugLog < Log.level {
		return
	}
	Log.Debug(a...)
}
func Debugf(format string, a ...interface{}) {
	if DebugLog < Log.level {
		return
	}
	Log.Debugf(format, a...)
}
func Info(a ...interface{}) {
	Log.Info(a...)
}
func Warn(a ...interface{}) {
	Log.Warn(a...)
}
func Error(a ...interface{}) {
	Log.Error(a...)
}
func Fatal(a ...interface{}) {
	Log.Fatal(a...)
}
func Infof(format string, a ...interface{}) {
	Log.Infof(format, a...)
}

func Println(a ...interface{}) {
	Log.Info(a...)
}
func Printf(format string, a ...interface{}) {
	Log.Printf(format, a...)
}
func Warnf(format string, a ...interface{}) {
	Log.Warnf(format, a...)
}
func Errorf(format string, a ...interface{}) {
	Log.Errorf(format, a...)
}
func Failed(format string, a ...interface{}) {
	Log.Errorf(format, a...)
}
func Fatalf(format string, a ...interface{}) {
	Log.Fatalf(format, a...)
}
func FileOpen(path string) (*os.File, error) {
	if fi, err := os.Stat(path); err == nil {
		if !fi.IsDir() {
			return nil, fmt.Errorf("open %s: not a directory", path)
		}
	} else if os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0766); err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}
	var currenttime = time.Now().Format("2006-01-02_15.04.05")
	logfile, err := os.OpenFile(path+currenttime+"_LOG.log", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	return logfile, nil
}

// Init deprecated, use InitLog instead
func Init(a ...interface{}) error {
	_, err := os.Stderr.WriteString("warning: use of deprecated Init. Use InitLog instead\n")
	if err != nil {
		return err
	}
	InitLog(InfoLog, a...)
	return nil
}
func InitLog(logLevel int, a ...interface{}) {
	writers := []io.Writer{}
	var logFile *os.File
	var err error

	if len(a) == 0 {
		writers = append(writers, io.Discard)
	} else {
		for _, o := range a {
			switch v := o.(type) {
			case string:
				logFile, err = FileOpen(v)
				if err != nil {
					fmt.Println("error: open log files failed")
					os.Exit(1)
				}
				writers = append(writers, logFile)
			case *os.File:
				writers = append(writers, v)
			default:
				fmt.Println("error: invalid log location")
				os.Exit(1)
			}
		}
	}

	fileAndStdoutWrite := io.MultiWriter(writers...)
	Log = New(fileAndStdoutWrite, "", log.LUTC|log.Ldate|log.Lmicroseconds, logLevel, logFile)
}

func GetLogFileSize() (int64, error) {
	f, e := Log.logFile.Stat()
	if e != nil {
		return 0, e
	}
	return f.Size(), nil
}
func GetMaxLogChangeInterval(maxLogSize int64) int64 {
	if maxLogSize != 0 {
		return (maxLogSize * BYTE_TO_MB)
	} else {
		return (DEFAULT_MAX_LOG_SIZE * BYTE_TO_MB)
	}
}
func CheckIfNeedNewFile() bool {
	logFileSize, err := GetLogFileSize()
	maxLogFileSize := GetMaxLogChangeInterval(0)
	if err != nil {
		return false
	}
	if logFileSize > maxLogFileSize {
		return true
	} else {
		return false
	}
}
func ClosePrintLog() error {
	var err error
	if Log.logFile != nil {
		err = Log.logFile.Close()
	}
	return err
}
