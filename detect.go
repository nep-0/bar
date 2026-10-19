package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/go-enry/go-enry/v2"
)

func detectLanguage(filename string, content []byte) (string, error) {
	if enry.IsBinary(content) {
		return "", fmt.Errorf("binary files cannot be run")
	}
	return resolveCandidates(enry.GetLanguages(filename, content))
}

func resolveCandidates(candidates []string) (string, error) {
	switch len(candidates) {
	case 0:
		return "", fmt.Errorf("could not detect a programming language")
	case 1:
		return candidates[0], nil
	default:
		sort.Strings(candidates)
		return "", fmt.Errorf("ambiguous language: %s", strings.Join(candidates, ", "))
	}
}
