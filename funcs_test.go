package main

import (
	"strings"
	"testing"
)

//TODO: write tests where retrieve values from map[string]struct{}
//TODO: write tests for func GetUniqueChars(s string) map[string]struct{}

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
