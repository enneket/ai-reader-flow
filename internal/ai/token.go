package ai

// Estimate estimates token count for a string using a simple ratio model.
// CJK characters are estimated at ~1.5 tokens each.
// ASCII characters are estimated at ~4 per token.
// This is a rough approximation suitable for budget planning, not billing accuracy.
func Estimate(text string) int {
    if text == "" {
        return 0
    }

    cjkCount := 0
    asciiCount := 0
    wsCount := 0

    for _, r := range text {
        switch {
        case isCJK(r):
            cjkCount++
        case r == ' ' || r == '\t' || r == '\n' || r == '\r':
            wsCount++
        default:
            asciiCount++
        }
    }

    // CJK: ~1.5 chars per token
    // ASCII: ~4 chars per token
    // Whitespace: ~5 chars per token (overhead)
    cjkTokens := float64(cjkCount) / 1.5
    asciiTokens := float64(asciiCount) / 4.0
    wsTokens := float64(wsCount) / 5.0

    return int(cjkTokens + asciiTokens + wsTokens)
}

// isCJK returns true if r is a CJK character
func isCJK(r rune) bool {
    return (r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified Ideographs
           (r >= 0x3000 && r <= 0x303F) || // CJK Symbols
           (r >= 0xFF00 && r <= 0xFFEF) || // Halfwidth/Fullwidth Forms
           (r >= 0x3040 && r <= 0x309F) || // Hiragana
           (r >= 0x30A0 && r <= 0x30FF) || // Katakana
           (r >= 0xAC00 && r <= 0xD7AF)    // Hangul Syllables
}