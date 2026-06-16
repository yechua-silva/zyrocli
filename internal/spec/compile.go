package spec

import (
	"fmt"
	"strings"
)

// EngramEntry represents a single Engram memory entry produced by Compile.
type EngramEntry struct {
	TopicKey string
	Content  string
}

// Compile translates a CIO struct into Engram topic keys and markdown content.
// It DOES NOT emit OpenAPI or protobuf — the output is designed for Engram
// traceability: each feature maps to its CIO, which maps to its Engram key,
// which maps to its implementation.
//
// changeName is the SDD change name used in the topic key prefix (e.g. "scheduler-harness").
func Compile(cio *CIO, changeName string) ([]EngramEntry, error) {
	if cio == nil {
		return nil, fmt.Errorf("compile: nil CIO")
	}

	component := cio.Contract.Name
	if component == "" {
		component = "unnamed"
	}
	component = strings.ToLower(strings.ReplaceAll(component, " ", "-"))

	content := cio.ToMarkdown()

	return []EngramEntry{
		{
			TopicKey: fmt.Sprintf("sdd/%s/cio-%s", changeName, component),
			Content:  content,
		},
	}, nil
}

// GenerateTopicKey produces a stable topic key for a given CIO and change name.
func GenerateTopicKey(cio *CIO, changeName string) string {
	component := cio.Contract.Name
	if component == "" {
		component = "unnamed"
	}
	component = strings.ToLower(strings.ReplaceAll(component, " ", "-"))
	return fmt.Sprintf("sdd/%s/cio-%s", changeName, component)
}
