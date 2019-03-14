package main

import (
	"bytes"
	"strings"
	"testing"
)

var gcb [][]byte
var gcs []string

func BenchmarkSplitToBytes(b *testing.B) {
	filename := "HP01.txt"
	text_dir := "./text/"

	var line_tokens [][]byte

	for i := 0; i < b.N; i++ {
		line_tokens = bytes.Split(get_filedata(text_dir, filename), []byte("\n"))
	}

	gcb = line_tokens
}

func BenchmarkSplitToStrings(b *testing.B) {
	filename := "HP01.txt"
	text_dir := "./text/"

	var line_tokens []string

	for i := 0; i < b.N; i++ {
		line_tokens = strings.Split(string(get_filedata(text_dir, filename)), "\n")
	}

	gcs = line_tokens
}
