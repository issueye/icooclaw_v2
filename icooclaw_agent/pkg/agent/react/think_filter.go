package react

import (
	"strings"
	"unicode/utf8"
)

const (
	thinkStartTag = "<think>"
	thinkEndTag   = "</think>"
)

type thinkExtraction struct {
	Visible   string
	Reasoning string
}

// extractThinkBlocks splits visible content and <think> reasoning from complete text payloads.
func extractThinkBlocks(content string) thinkExtraction {
	if content == "" {
		return thinkExtraction{}
	}

	var visible strings.Builder
	var reasoningParts []string
	remaining := content
	for {
		start := strings.Index(remaining, thinkStartTag)
		if start < 0 {
			visible.WriteString(remaining)
			break
		}

		visible.WriteString(remaining[:start])
		remaining = remaining[start+len(thinkStartTag):]

		end := strings.Index(remaining, thinkEndTag)
		if end < 0 {
			break
		}

		thought := strings.TrimSpace(remaining[:end])
		if thought != "" {
			reasoningParts = append(reasoningParts, thought)
		}
		remaining = remaining[end+len(thinkEndTag):]
	}

	return thinkExtraction{
		Visible:   strings.TrimSpace(visible.String()),
		Reasoning: strings.TrimSpace(strings.Join(reasoningParts, "\n\n")),
	}
}

func stripThinkBlocks(content string) string {
	return extractThinkBlocks(content).Visible
}

// thinkStreamFilter incrementally splits visible content and reasoning from streamed chunks.
type thinkStreamFilter struct {
	buffer               string
	inThink              bool
	visiblePassthrough   strings.Builder
	reasoningPassthrough strings.Builder
}

func newThinkStreamFilter() *thinkStreamFilter {
	return &thinkStreamFilter{}
}

func (f *thinkStreamFilter) Push(chunk string) thinkExtraction {
	if chunk == "" {
		return thinkExtraction{}
	}

	f.buffer += chunk
	for {
		if f.inThink {
			end := strings.Index(f.buffer, thinkEndTag)
			if end < 0 {
				safe := flushSafePrefix(f.buffer, len(thinkEndTag)-1)
				f.reasoningPassthrough.WriteString(safe)
				f.buffer = f.buffer[len(safe):]
				break
			}
			f.reasoningPassthrough.WriteString(f.buffer[:end])
			f.buffer = f.buffer[end+len(thinkEndTag):]
			f.inThink = false
			continue
		}

		start := strings.Index(f.buffer, thinkStartTag)
		if start < 0 {
			safe := flushSafePrefix(f.buffer, len(thinkStartTag)-1)
			f.visiblePassthrough.WriteString(safe)
			f.buffer = f.buffer[len(safe):]
			break
		}

		f.visiblePassthrough.WriteString(f.buffer[:start])
		f.buffer = f.buffer[start+len(thinkStartTag):]
		f.inThink = true
	}

	return f.take()
}

func (f *thinkStreamFilter) Flush() thinkExtraction {
	if f.buffer != "" {
		if f.inThink {
			f.reasoningPassthrough.WriteString(f.buffer)
		} else {
			f.visiblePassthrough.WriteString(f.buffer)
		}
	}
	f.buffer = ""
	return f.take()
}

func (f *thinkStreamFilter) take() thinkExtraction {
	result := thinkExtraction{
		Visible:   f.visiblePassthrough.String(),
		Reasoning: strings.TrimSpace(f.reasoningPassthrough.String()),
	}
	f.visiblePassthrough.Reset()
	f.reasoningPassthrough.Reset()
	return result
}

func flushSafePrefix(s string, reserve int) string {
	if reserve <= 0 || len(s) <= reserve {
		return ""
	}

	end := len(s) - reserve
	for end > 0 && end < len(s) && !utf8.RuneStart(s[end]) {
		end--
	}

	return s[:end]
}

func joinReasoningParts(parts ...string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	return strings.Join(filtered, "\n\n")
}
