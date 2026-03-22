package logging

import (
	"log"
	"sort"
	"strings"
)

type StdLogger struct{}

func NewStdLogger() StdLogger {
	return StdLogger{}
}

func (logger StdLogger) Info(message string, fields map[string]string) {
	log.Printf("INFO %s %s", message, formatFields(fields))
}

func (logger StdLogger) Error(message string, fields map[string]string) {
	log.Printf("ERROR %s %s", message, formatFields(fields))
}

func formatFields(fields map[string]string) string {
	if len(fields) == 0 {
		return ""
	}

	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+fields[key])
	}

	return strings.Join(parts, " ")
}
