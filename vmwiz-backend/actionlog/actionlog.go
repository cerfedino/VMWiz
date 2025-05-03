package actionlog

import (
	"fmt"
	"log"
	"os"
)

func Printf(uuid string, format string, v ...any) {
	line := fmt.Sprintf(format, v...)
	Println(uuid, line)
}

func Println(uuid string, line string) {
	log.Println(line)
	if uuid == "testing" {
		return
	}
	// write to log file uuid.log
	file, err := os.OpenFile("logs/"+uuid+".log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Failed to open log file: %v", err)
		return
	}
	defer file.Close()

	if _, err := file.WriteString(line + "\n"); err != nil {
		log.Printf("Failed to write to log file: %v", err)
		return
	}
}
