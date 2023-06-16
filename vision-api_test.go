// Copyright 2021-2023, Matthew Winter
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"io"
	"reflect"
	"testing"
)

func Test_handleIgnoreFile(t *testing.T) {

	someContent := bytes.NewBufferString("*.jpg\n*.bmp\n./examples/**")
	noContent := bytes.NewBufferString("")
	malformed := bytes.NewBufferString("   *.jpg\n*.bmp \n  ./examples/**")

	tests := []struct {
		name string
		file io.Reader
		want []string
	}{
		{
			name: "some values",
			file: someContent,
			want: []string{"*.jpg", "*.bmp", "./examples/**"},
		},
		{
			name: "one",
			file: noContent,
			want: []string{},
		},
		{
			name: "whitespaced",
			file: malformed,
			want: []string{"*.jpg", "*.bmp", "./examples/**"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handleIgnoreFile(tt.file); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleIgnoreFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isIgnorableFile(t *testing.T) {

	type args struct {
		fileName   string
		ignoreList []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "ok matches",
			args: args{
				fileName:   "aloha.jpg",
				ignoreList: []string{"*.jpg"},
			},
			want: true,
		},
		{
			name: "not matches",
			args: args{
				fileName:   "aloha.jpg",
				ignoreList: []string{"*.bmp"},
			},
			want: false,
		},
		{
			name: "not matches because in dir",
			args: args{
				fileName:   "./examples/aloha.jpg",
				ignoreList: []string{"*.bmp"},
			},
			want: false,
		},
		{
			name: "ok matches in dir",
			args: args{
				fileName:   "./examples/aloha.jpg",
				ignoreList: []string{"./examples/*.jpg"},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsIgnorableFile(tt.args.fileName, tt.args.ignoreList); got != tt.want {
				t.Errorf("isIgnorableFile() = %v, want %v", got, tt.want)
			}
		})
	}
}
