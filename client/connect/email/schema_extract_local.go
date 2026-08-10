package main

import (
	"encoding/json"
	"regexp"
	"strings"
)

var jsonObjectBoundaryPatternLocal = regexp.MustCompile(`[{}]`)

func schemaJSONCandidateLocal(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	if strings.HasPrefix(raw, "{") && strings.HasSuffix(raw, "}") {
		return raw, true
	}
	return extractEmbeddedJSONObjectLocal(raw)
}

func extractEmbeddedJSONObjectLocal(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}

	boundaries := jsonObjectBoundaryPatternLocal.FindAllStringIndex(raw, -1)
	if len(boundaries) < 2 {
		return "", false
	}

	starts := make([]int, 0, len(boundaries))
	ends := make([]int, 0, len(boundaries))
	for _, boundary := range boundaries {
		token := raw[boundary[0]:boundary[1]]
		switch token {
		case "{":
			starts = append(starts, boundary[0])
		case "}":
			ends = append(ends, boundary[0])
		}
	}
	if len(starts) == 0 || len(ends) == 0 {
		return "", false
	}

	for _, start := range starts {
		for i := len(ends) - 1; i >= 0; i-- {
			end := ends[i]
			if end <= start {
				continue
			}
			candidate := strings.TrimSpace(raw[start : end+1])
			if candidate == "" {
				continue
			}
			if json.Valid([]byte(candidate)) {
				return candidate, true
			}
		}
	}
	return "", false
}
