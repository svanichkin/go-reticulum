package main

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

func globSerialPatterns(patterns []string) []string {
	seen := map[string]struct{}{}
	var ports []string
	for _, pat := range patterns {
		matches, _ := filepath.Glob(pat)
		for _, m := range matches {
			if _, ok := seen[m]; ok {
				continue
			}
			seen[m] = struct{}{}
			ports = append(ports, m)
		}
	}
	sort.Strings(ports)
	return ports
}

func serialPortDisplayLabel(port string) string {
	port = strings.TrimSpace(port)
	if port == "" {
		return ""
	}
	if strings.Contains(port, "/serial/by-id/") {
		base := path.Base(port)
		if base != "" && base != "." && base != "/" {
			return fmt.Sprintf("%s (%s)", port, base)
		}
	}
	return port
}
