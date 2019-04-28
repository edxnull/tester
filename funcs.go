package main

import (
	"strings"
)

// we need to make sure we can add:
// the possibility to create predefined rules on the fly in real time
func GetUniqueWords(s []string) map[string]struct{} {
	mk := make(map[string]struct{})
	for i := 0; i < len(s); i++ {
		words := strings.Split(s[i], " ")
		for _, w := range words {
			if HasNonAlpha(w) {
				trim := strings.ToLower(strings.Trim(w, ",.\n\r\\/\"'-;%^$#*@(!?)_-+=:<>[]{}~|"))
				if !AllNonAlpha(w) {
					if _, ok := mk[trim]; !ok {
						mk[trim] = struct{}{}
					}
				}
			} else {
				if w != "" {
					mk[w] = struct{}{}
				}
			}
		}
		words = nil
	}
	return mk
}

func HasNonAlpha(str string) bool {
	for _, c := range []byte(str) {
		if !IsAlpha(c) {
			return true
		}
	}
	return false
}

func AllNonAlpha(str string) bool {
	for _, c := range []byte(str) {
		if IsAlpha(c) {
			return false
		}
	}
	return true
}

func HasCapitalLetter(str string) bool {
	for _, c := range []byte(str) {
		if IsCapital(c) {
			return true
		}
	}
	return false
}

func IsCapital(c byte) bool {
	return (c >= byte('A')) && (c <= byte('Z'))
}

func IsAlpha(c byte) bool {
	return (c >= byte('A')) && (c <= byte('z'))
}
