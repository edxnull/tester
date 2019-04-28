package main

import (
	"strings"
	"testing"
)

//TODO: write tests where retrieve values from map[string]struct{}

var tests = []struct {
	input  string
	output []string
}{
	{input: "", output: []string{""}},
	{input: " ", output: []string{""}},
	{input: "one", output: []string{"one"}},
	{input: "one..", output: []string{"one"}},
	{input: "free'd", output: []string{"free'd"}},
	{input: "l'amour", output: []string{"l'amour"}},
	{input: "forget-me-not", output: []string{"forget-me-not"}},
	{input: "one!!!---", output: []string{"one"}},
	{input: "..one..", output: []string{"one"}},
	{input: "one two", output: []string{"one", "two"}},
	{input: "!one--! two-", output: []string{"one", "two"}},
	{input: "one@(*#... two..@* three#^##@^", output: []string{"one", "two", "three"}},
	{input: "one@(*#... two..@* three aga#^##@^", output: []string{"one", "two", "three", "aga"}},
}

func TestGetUniqueWords(t *testing.T) {
	for _, tt := range tests {
		words := GetUniqueWords([]string{tt.input})
		if len(words) > len(tt.output) {
			t.Errorf("TestGetUniqueWords failed: words len: %d, tt.output len: %d", len(words), len(tt.output))
			break
		}
		for i := 0; i < len(words); i++ {
			for _, out := range strings.Split(tt.output[i], " ") {
				if _, ok := words[out]; !ok {
					t.Errorf("TestGetUniqueWords failed: (%s) in words[out] (%t)", out, ok)
				}
			}
		}
	}
}

//go:noinline
func BenchmarkGetUniqueWords(b *testing.B) {
	var words map[string]struct{}
	for i := 0; i < b.N; i++ {
		words = GetUniqueWords([]string{
			"one",
			"one two three",
			"one duck is... going.. home!!! not too far@* ahead... forever thy be here",
			"here is another string to bring back home",
			"and another right here at that!!!!",
			"lalalalla more lalalal tralalala",
		})
	}
	_ = words
}

var test_wrapped_lines = []struct {
	input    []string
	expected int32
}{
	{
		input:    []string{"one"},
		expected: 1,
	},
	{
		input:    []string{"one", "two"},
		expected: 2,
	},
	{
		input: []string{
			"one two three four five six seven eight",
		},
		expected: 1,
	},
	{
		input: []string{
			"five six seven eight nine",
			"blaaah blaaah blaaah",
		},
		expected: 2,
	},
	{
		input: []string{
			"one two three four five six seven eight",
			"five six seven eight nine",
			"blaaah blaaah blaaah",
		},
		expected: 3,
	},
}

func TestNumWrappedLines(t *testing.T) {
	var result int32
	for _, tt := range test_wrapped_lines {
		result = NumWrappedLines(tt.input, 500, 12) // len and size and offset
		if result != tt.expected {
			t.Errorf("TestNumWrappedLines failed: got (%d) expected (%d) ", result, tt.expected)
		}
	}
}

//go:noinline
func BenchmarkNumWrappedLines(b *testing.B) {
	var result int32
	for i := 0; i < b.N; i++ {
		result = NumWrappedLines([]string{
			"Roads go ever ever on,",
			"Over rock and under tree,",
			"By caves where never sun has shone,",
			"By streams that never find the sea;",
			"Over snow by winter sown,",
			"And through the merry flowers of June,",
			"Over grass and over stone,",
			"And under mountains in the moon.",

			"Roads go ever ever on",
			"Under cloud and under star,",
			"Yet feet that wandering have gone",
			"Turn at last to home afar.",
			"Eyes that fire and sword have seen",
			"And horror in the halls of stone",
			"Look at last on meadows green",
			"And trees and hills they long have known.",
			"And trees and hills they long have known." + "aaaaaaaaa bbbbbbbbb ccccccccc ddddddddd",
			"And trees and hills they long have known." + "aaaaaaaaa bbbbbbbbb ccccccccc ddddddddd",
			"And trees and hills they long have known." + "aaaaaaaaa bbbbbbbbb ccccccccc ddddddddd",
			"And trees and hills they long have known." + "aaaaaaaaa bbbbbbbbb ccccccccc dddddddddasdasdasd",
			"And trees and hills they long have known." + "aaaaaaaaa bbbbbbbbb ccccccccc dddddddddasdasdasd",
		}, 500, 12)
	}
	_ = result
}

func TestDoWrapLines(t *testing.T) {
	test := []struct {
		input    string
		expected []string
	}{
		{
			input:    "one two three",
			expected: []string{"one two three"},
		},
		{
			input:    "one two three four five",
			expected: []string{"one two three four five"},
		},
	}
	var result []string
	for _, tt := range test {
		result = DoWrapLines(tt.input, 500, 12)
		if len(result) != len(tt.expected) {
			t.Errorf("TestDoWrapLines failed length: got len(%s) expected len(%s) ", result, tt.expected)
		}
		for i := 0; i < len(result); i++ {
			if result[i] != tt.expected[i] {
				t.Errorf("TestDoWrapLines failed string: got (%s) expected (%s) ", result, tt.expected)
			}
		}
	}
	_ = result
}

//go:noinline
func BenchmarkDoWrapLine(b *testing.B) { // add sub benchmarks here
	var result []string
	for i := 0; i < b.N; i++ {
		result = DoWrapLines("And trees and hills they known.aaaaaaaaa bbbbbbbbb ccccccccc ddddddddd", 500, 12)
	}
	_ = result
}
