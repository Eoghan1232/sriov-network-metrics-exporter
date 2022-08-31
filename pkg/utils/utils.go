package utils

import (
	"log"
	"path/filepath"
	"strings"
)

// comma-separated string flag type
type StringListFlag []string

func (list *StringListFlag) String() string {
	return strings.Join(*list, ",")
}

func (list *StringListFlag) Set(val string) error {
	*list = strings.Split(val, ",")

	for i := range *list {
		(*list)[i] = strings.TrimSpace((*list)[i])
	}

	return nil
}

func ResolvePath(path *string) {
	cleanPath, err := filepath.EvalSymlinks(filepath.Clean(*path))
	if err != nil {
		log.Fatalf("Unsafe or invalid path specified, Error: %v", err)
	} else {
		*path = cleanPath
	}
}
