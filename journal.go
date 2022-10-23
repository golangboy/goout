package goout

import (
	"log"
	"os"
	"runtime"
	"strconv"
)

func init() {
	fs, _ := os.Create("journal.log")
	log.SetOutput(fs)
	log.SetFlags(log.LstdFlags)
}
func LogInfo(v any) {
	_, file, line, _ := runtime.Caller(1)
	log.Print("[LogInfo]")
	log.Print(file + ":" + strconv.Itoa(line))
	log.Println(v)
	log.Println()
	log.Println()
}

func LogError(v any) {
	_, file, line, _ := runtime.Caller(1)
	log.Print("[LogError]")
	log.Print(file + ":" + strconv.Itoa(line))
	log.Println(v)
	log.Println()
	log.Println()
}
