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
