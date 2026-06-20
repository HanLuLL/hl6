package helpers

import "unicode/utf8"

// TruncateRunes 按 rune 截断字符串，超出时追加 "..." 后缀。
func TruncateRunes(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// SnippetAroundIndex 在 rune 索引 idx 处提取关键词/匹配周围的上下文片段。
func SnippetAroundIndex(s string, idx, matchLen, before, after int) string {
	runes := []rune(s)
	if len(runes) == 0 || idx < 0 {
		return ""
	}
	if idx >= len(runes) {
		idx = len(runes) - 1
	}
	start := idx - before
	if start < 0 {
		start = 0
	}
	end := idx + matchLen + after
	if end > len(runes) {
		end = len(runes)
	}
	return string(runes[start:end])
}

// SnippetAroundByteIndex 在 UTF-8 字节索引处提取片段。
func SnippetAroundByteIndex(target string, byteIdx, matchByteLen, before, after int) string {
	if byteIdx < 0 || byteIdx >= len(target) {
		return ""
	}
	prefix := target[:byteIdx]
	runeIdx := utf8.RuneCountInString(prefix)
	matchRunes := utf8.RuneCountInString(target[byteIdx : byteIdx+matchByteLen])
	return SnippetAroundIndex(target, runeIdx, matchRunes, before, after)
}
