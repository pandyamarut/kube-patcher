package kubernetes

import (
	"os"
)

// Getenv will return enviornment variables
func Getenv() map[string]string {
	return map[string]string{
		"namespace":    os.Getenv("NAMESPACE"),
		"resourceName": os.Getenv("APPLICATION_NAME"),
	}
}
