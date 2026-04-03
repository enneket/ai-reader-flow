package ai

import "testing"

func TestEstimate(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantMin int
        wantMax int
    }{
        {
            name:    "empty string",
            input:   "",
            wantMin: 0,
            wantMax: 0,
        },
        {
            name:    "ASCII text",
            input:   "Hello world this is a test",
            wantMin: 5,
            wantMax: 25,
        },
        {
            name:    "CJK text",
            input:   "这是一段测试文本用于测试Token估算",
            wantMin: 10,
            wantMax: 30,
        },
        {
            name:    "mixed content",
            input:   "Hello 你好 World 世界",
            wantMin: 4,
            wantMax: 16,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := Estimate(tt.input)
            if got < tt.wantMin || got > tt.wantMax {
                t.Errorf("Estimate(%q) = %d, want between %d and %d", tt.input, got, tt.wantMin, tt.wantMax)
            }
        })
    }
}