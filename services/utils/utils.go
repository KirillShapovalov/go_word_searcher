package utils

import (
	"log"
)

func HandleDeferClose(resourceName string, closer func() error) {
	err := closer()
	if err != nil {
		log.Printf("warning: failed to close %s: %v", resourceName, err)
	}
}
