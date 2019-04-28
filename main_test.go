package main

import (
	"bytes"
	"strings"
	"testing"
)

var br [][]byte
var sr []string

func BenchmarkSplitToBytes(b *testing.B) {
	var r [][]byte
	filename := "HP01.txt"
	text_dir := "./text/"

	text := get_filedata(text_dir, filename)

	for i := 0; i < b.N; i++ {
		r = bytes.Split(text, []byte("\n"))
	}
	br = r
}

func BenchmarkSplitToStrings(b *testing.B) {
	var r []string
	filename := "HP01.txt"
	text_dir := "./text/"

	text := string(get_filedata(text_dir, filename))

	for i := 0; i < b.N; i++ {
		r = strings.Split(text, "\n")
	}
	sr = r
}

//go:noinline
func BenchmarkEaseInQuad(b *testing.B) {
	var out float32
	bb := float32(0)
	d := float32(30)
	c := float32(d - bb)
	t := float32(10)

	for i := 0; i < b.N; i++ {
		out = EaseInQuad(bb, d, c, t)
		t += 1
	}
	_ = out
}

//go:noinline
func BenchmarkEaseOutQuad(b *testing.B) {
	var in float32
	bb := float32(0)
	d := float32(30)
	c := float32(d - bb)
	t := float32(10)

	for i := 0; i < b.N; i++ {
		in = EaseOutQuad(bb, d, c, t)
		t += 1
	}
	_ = in
}

//go:noinline
func BenchmarkEaseInOutQuad(b *testing.B) {
	var in float32
	bb := float32(0)
	d := float32(30)
	c := float32(d - bb)
	t := float32(10)

	for i := 0; i < b.N; i++ {
		in = EaseInOutQuad(bb, d, c, t)
		t += 1
	}
	_ = in
}

func TestCustomTrim(t *testing.T) {
	// TODO: fix test fail on "...word"
	str := []string{"one,", "two...", "three'd", "four", "five", "six-o-clock"}
	want := []string{"one", "two", "three", "four", "five", "six-o-clock"}
	trim := make([]int, len(str))

	for i := 0; i < len(str); i++ {
		for j := 0; j < len(str[i]); j++ {
			if !(str[i][j] >= byte('A') && str[i][j] <= byte('z')) && str[i][j] != byte('-') {
				trim[i] = j
				break
			}
			if j == len(str[i])-1 {
				trim[i] = j + 1
			}
		}
	}

	for i, s := range str {
		if s[:trim[i]] != want[i] {
			t.Errorf("custom trim failure (got: %s, want: %s)", s[:trim[i]], want[i])
		}
	}
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
