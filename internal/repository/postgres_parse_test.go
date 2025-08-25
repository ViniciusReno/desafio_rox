package repository

import (
	"testing"
)

func TestParseLine(t *testing.T) {
	tests := []struct {
		line   string
		ticker string
		price  float64
		qty    float64
		hour   int
		minute int
		second int
	}{
		{
			line:   "2025-08-22;DIIF31F32;0;0,110;200;090000034;10;1;2025-08-22;114;114",
			ticker: "DIIF31F32",
			price:  0.110,
			qty:    200,
			hour:   9,
			minute: 0,
			second: 0,
		},
		{
			line:   "2024-05-05;PETR4;10.5;100;12:00:00",
			ticker: "PETR4",
			price:  10.5,
			qty:    100,
			hour:   12,
			minute: 0,
			second: 0,
		},
	}
	for _, tt := range tests {
		ticker, price, qty, tm, ok, err := parseLine(tt.line)
		if err != nil || !ok {
			t.Fatalf("parseLine error for %q: %v", tt.line, err)
		}
		if ticker != tt.ticker || price != tt.price || qty != tt.qty {
			t.Fatalf("unexpected parse result for %q", tt.line)
		}
		if tm.Hour() != tt.hour || tm.Minute() != tt.minute || tm.Second() != tt.second {
			t.Fatalf("unexpected time for %q: %v", tt.line, tm)
		}
	}

	if _, _, _, _, ok, _ := parseLine("bad;data"); ok {
		t.Fatalf("expected invalid line to be skipped")
	}
}
