// Copyright 2016 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.
//
// Author: Tamir Duberstein (tamird@gmail.com)

package pb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"strconv"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
)

var typeProtoMessage = reflect.TypeOf((*proto.Message)(nil)).Elem()

// JSONPb is a Marshaler which marshals/unmarshals into/from JSON
// with the "github.com/gogo/protobuf/jsonpb".
// It supports fully functionality of protobuf unlike JSONBuiltin.
type JSONPb jsonpb.Marshaler

// ContentType always returns "application/json".
func (*JSONPb) ContentType() string {
	return "application/json"
}

// Marshal marshals "v" into JSON
func (j *JSONPb) Marshal(v interface{}) (data []byte, err error) {
	return jsonMarshal(v, true)
}

// Unmarshal unmarshals JSON "data" into "v"
func (j *JSONPb) Unmarshal(data []byte, v interface{}) error {
	if str, err := strconv.Unquote(string(data)); err == nil {
		return json.Unmarshal([]byte(str), v)
	}

	return json.Unmarshal(data, v)
}

// NewDecoder returns a Decoder which reads JSON stream from "r".
func (j *JSONPb) NewDecoder(r io.Reader) gwruntime.Decoder {
	return gwruntime.DecoderFunc(func(v interface{}) error {
		if pb, ok := v.(proto.Message); ok {
			if data, err := ioutil.ReadAll(r); err == nil {
				return j.Unmarshal(data, pb)
			}
			return jsonpb.Unmarshal(r, pb)
		}
		return fmt.Errorf("unexpected type %T does not implement %s", v, typeProtoMessage)
	})
}

// NewEncoder returns an Encoder which writes JSON stream into "w".
func (j *JSONPb) NewEncoder(w io.Writer) gwruntime.Encoder {
	return gwruntime.EncoderFunc(func(v interface{}) error {
		if pb, ok := v.(proto.Message); ok {
			marshalFn := (*jsonpb.Marshaler)(j).Marshal
			return marshalFn(w, pb)
		}
		return fmt.Errorf("unexpected type %T does not implement %s", v, typeProtoMessage)
	})
}

func jsonMarshal(v interface{}, safeEncoding bool) ([]byte, error) {
	b, err := json.Marshal(v)
	if safeEncoding {
		b = bytes.Replace(b, []byte("\\u003c"), []byte("<"), -1)
		b = bytes.Replace(b, []byte("\\u003e"), []byte(">"), -1)
		b = bytes.Replace(b, []byte("\\u0026"), []byte("&"), -1)
	}
	return b, err
}
