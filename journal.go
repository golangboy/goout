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
	_, file2, line2, _ := runtime.Caller(2)
	_, file3, line3, _ := runtime.Caller(3)
	_, file4, line4, _ := runtime.Caller(4)
	_, file5, line5, _ := runtime.Caller(5)
	log.Print("[LogError]")
	log.Print(file + ":" + strconv.Itoa(line))
	log.Println(file2 + ":" + strconv.Itoa(line2))
	log.Println(file3 + ":" + strconv.Itoa(line3))
	log.Println(file4 + ":" + strconv.Itoa(line4))
	log.Println(file5 + ":" + strconv.Itoa(line5))
	log.Println(v)
	log.Println()
	log.Println()
}
