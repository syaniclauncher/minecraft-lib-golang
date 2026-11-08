package main

import (
	"reflect"
	"testing"
)

func TestParseJVMArguments(t *testing.T) {
	input := `-javaagent:"E:\Desktop\minecraft lib golang\examples\launcher\authlib-injector-1.2.8.jar"=ely.by -Xmx2G`
	want := []string{
		`-javaagent:E:\Desktop\minecraft lib golang\examples\launcher\authlib-injector-1.2.8.jar=ely.by`,
		"-Xmx2G",
	}

	got, err := parseJVMArguments(input)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseJVMArguments() = %#v, want %#v", got, want)
	}
}

func TestParseJVMArgumentsUnmatchedQuote(t *testing.T) {
	if _, err := parseJVMArguments(`-javaagent:"path with spaces.jar`); err == nil {
		t.Fatal("parseJVMArguments() error = nil, want unmatched quote error")
	}
}
