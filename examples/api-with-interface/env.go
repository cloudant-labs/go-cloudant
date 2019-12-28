/**
 * env
 * - read environment variables
 */

package main

import (
	"bufio"
	"io"
	"os"
	"strings"
)

// Getenv allows reading environment variable from either environment or .env file
func Getenv(name string) string {
	value := os.Getenv(name)

	if value != "" {
		return value
	}

	f, err := os.Open(".env")
	if err != nil {
		return ""
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')

		// check if the line has = sign
		// and process the line. Ignore the rest.
		if equal := strings.Index(line, "="); equal >= 0 {
			if key := strings.TrimSpace(line[:equal]); len(key) > 0 {
				value := ""
				if len(line) > equal {
					value = strings.TrimSpace(line[equal+1:])
				}
				// assign the config map
				if key == name {
					return value
				}
			}
		}
		if err == io.EOF {
			return ""
		}
		if err != nil {
			return ""
		}
	}

}
