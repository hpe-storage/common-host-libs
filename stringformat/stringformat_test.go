// (c) Copyright 2018 Hewlett Packard Enterprise Development LP

package stringformat

import (
	"testing"
)

func TestFixedLengthString(t *testing.T) {
	var value1 uint64
	var value2 uint64
	var value3 uint64
	value1 = 0
	value2 = 1024
	value3 = 16384
	type args struct {
		length int
		value  interface{}
		align  AlignmentType
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"test-1", args{12, "TestVolume123456789AB", LeftAlign}, "TestVolume12"},
		{"test-2", args{5, "HelloWorld", LeftAlign}, "Hello"},
		{"test-3", args{9, "This is a new world", LeftAlign}, "This is a"},
		{"test-4", args{7, "Foo", RightAlign}, "    Foo"},
		{"test-5", args{7, "Bar", CenterAlign}, "  Bar  "},
		{"test-6", args{7, true, CenterAlign}, " true  "},
		{"test-6.1", args{7, false, CenterAlign}, " false "},
		{"test-7", args{6, value1, CenterAlign}, "  0   "},
		{"test-8", args{8, value2, CenterAlign}, "  1024  "},
		{"test-9", args{10, value3, CenterAlign}, "  16384   "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FixedLengthString(tt.args.length, tt.args.value, tt.args.align); got != tt.want {
				t.Errorf("FixedLengthString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStringLookup(t *testing.T) {
	type args struct {
		input interface{}
		value string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"test-1", args{"foo", "foo"}, true},
		{"test-2", args{"foo", "bar"}, false},
		{"test-3", args{[]string{"foo", "bar", "hello"}, "bar"}, true},
		{"test-3", args{[]string{"foo", "bar", "hello"}, "test"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StringLookup(tt.args.input, tt.args.value); got != tt.want {
				t.Errorf("StringLookup() = %v, want %v", got, tt.want)
			}
		})
	}
}
