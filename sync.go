package sync

import (
	"log"
	"os"
)

func logf(format string, v ...interface{}) {
	if os.Getenv("DEBUG_SYNC") != "" {
		log.Printf(format, v...)
	}
}
