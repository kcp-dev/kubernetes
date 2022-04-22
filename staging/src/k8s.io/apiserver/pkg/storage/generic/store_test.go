/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package generic

import (
	"reflect"
	"testing"

	"k8s.io/apiserver/pkg/apis/example"
)

func Test_decodeContinue(t *testing.T) {
	type args struct {
		continueValue string
		keyPrefix     string
	}
	tests := []struct {
		name        string
		args        args
		wantFromKey string
		wantRv      int64
		wantErr     bool
	}{
		{name: "valid", args: args{continueValue: encodeContinueOrDie("meta.k8s.io/v1", 1, "key"), keyPrefix: "/test/"}, wantRv: 1, wantFromKey: "/test/key"},
		{name: "root path", args: args{continueValue: encodeContinueOrDie("meta.k8s.io/v1", 1, "/"), keyPrefix: "/test/"}, wantRv: 1, wantFromKey: "/test/"},

		{name: "empty version", args: args{continueValue: encodeContinueOrDie("", 1, "key"), keyPrefix: "/test/"}, wantErr: true},
		{name: "invalid version", args: args{continueValue: encodeContinueOrDie("v1", 1, "key"), keyPrefix: "/test/"}, wantErr: true},

		{name: "path traversal - parent", args: args{continueValue: encodeContinueOrDie("meta.k8s.io/v1", 1, "../key"), keyPrefix: "/test/"}, wantErr: true},
		{name: "path traversal - local", args: args{continueValue: encodeContinueOrDie("meta.k8s.io/v1", 1, "./key"), keyPrefix: "/test/"}, wantErr: true},
		{name: "path traversal - double parent", args: args{continueValue: encodeContinueOrDie("meta.k8s.io/v1", 1, "./../key"), keyPrefix: "/test/"}, wantErr: true},
		{name: "path traversal - after parent", args: args{continueValue: encodeContinueOrDie("meta.k8s.io/v1", 1, "key/../.."), keyPrefix: "/test/"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFromKey, gotRv, err := decodeContinue(tt.args.continueValue, tt.args.keyPrefix)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeContinue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotFromKey != tt.wantFromKey {
				t.Errorf("decodeContinue() gotFromKey = %v, want %v", gotFromKey, tt.wantFromKey)
			}
			if gotRv != tt.wantRv {
				t.Errorf("decodeContinue() gotRv = %v, want %v", gotRv, tt.wantRv)
			}
		})
	}
}

func Test_growSlice(t *testing.T) {
	type args struct {
		initialCapacity int
		v               reflect.Value
		maxCapacity     int
		sizes           []int
	}
	tests := []struct {
		name string
		args args
		cap  int
	}{
		{
			name: "empty",
			args: args{v: reflect.ValueOf([]example.Pod{})},
			cap:  0,
		},
		{
			name: "no sizes",
			args: args{v: reflect.ValueOf([]example.Pod{}), maxCapacity: 10},
			cap:  10,
		},
		{
			name: "above maxCapacity",
			args: args{v: reflect.ValueOf([]example.Pod{}), maxCapacity: 10, sizes: []int{1, 12}},
			cap:  10,
		},
		{
			name: "takes max",
			args: args{v: reflect.ValueOf([]example.Pod{}), maxCapacity: 10, sizes: []int{8, 4}},
			cap:  8,
		},
		{
			name: "with existing capacity above max",
			args: args{initialCapacity: 12, maxCapacity: 10, sizes: []int{8, 4}},
			cap:  12,
		},
		{
			name: "with existing capacity below max",
			args: args{initialCapacity: 5, maxCapacity: 10, sizes: []int{8, 4}},
			cap:  8,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.initialCapacity > 0 {
				tt.args.v = reflect.ValueOf(make([]example.Pod, 0, tt.args.initialCapacity))
			}
			// reflection requires that the value be addressible in order to call set,
			// so we must ensure the value we created is available on the heap (not a problem
			// for normal usage)
			if !tt.args.v.CanAddr() {
				x := reflect.New(tt.args.v.Type())
				x.Elem().Set(tt.args.v)
				tt.args.v = x.Elem()
			}
			growSlice(tt.args.v, tt.args.maxCapacity, tt.args.sizes...)
			if tt.cap != tt.args.v.Cap() {
				t.Errorf("Unexpected capacity: got=%d want=%d", tt.args.v.Cap(), tt.cap)
			}
		})
	}
}
