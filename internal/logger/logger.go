package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
)

var Verbose bool

func init() {
	log.SetFlags(log.Ldate | log.Ltime)
	log.SetOutput(os.Stdout)
}

func Event(kind string, fields ...string) {
	if !Verbose {
		return
	}
	var parts []string
	for i := 0; i+1 < len(fields); i += 2 {
		parts = append(parts, fmt.Sprintf("%s=%s", fields[i], fields[i+1]))
	}
	if len(parts) == 0 {
		log.Printf("%s", kind)
		return
	}
	log.Printf("%s %s", kind, strings.Join(parts, " "))
}
