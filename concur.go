package concur

import (
	"log"
	"os"
)

func logf(format string, v ...interface{}) {
	if os.Getenv("DEBUG_CONCUR") != "" {
		log.Printf(format, v...)
	}
}
