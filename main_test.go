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
func BenchmarkEaseInQuart_v1(b *testing.B) {
    var out float32
	bb := float32(0)
	d := float32(30)
	c := float32(d - bb)
    t := float32(10)

    for i := 0; i < b.N; i++ {
        out = EaseInQuart(bb, d, c, t)
    }
    _ = out
}

//go:noinline
func BenchmarkEaseOutQuart_v1(b *testing.B) {
    var in float32
	bb := float32(0)
	d := float32(30)
	c := float32(d - bb)
    t := float32(10)

    for i := 0; i < b.N; i++ {
        in = EaseOutQuart(bb, d, c, t)
    }
    _ = in
}
