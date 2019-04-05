package main

import (
	"bytes"
    "errors"
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
    var r[]string
	filename := "HP01.txt"
	text_dir := "./text/"

    text := string(get_filedata(text_dir, filename))

	for i := 0; i < b.N; i++ {
		r = strings.Split(text, "\n")
	}
    sr = r
}
