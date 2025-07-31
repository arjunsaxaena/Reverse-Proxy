package logger

import (
	"log"
	"os"
)

var (
	infoLogger  = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	fatalLogger = log.New(os.Stderr, "FATAL: ", log.Ldate|log.Ltime|log.Lshortfile)
)

func Info(v ...interface{}) {
	infoLogger.Println(v...)
}

func Fatal(v ...interface{}) {
	fatalLogger.Fatalln(v...)
} 