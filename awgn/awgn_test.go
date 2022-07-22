// Copyright 2020 Justine Alexandra Roberts Tunney
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dsp_test

import (
	"encoding/binary"
	"os"
	"testing"
	"unsafe"

	"github.com/pidato/audio/awgn"
)

func TestAWGN(t *testing.T) {
	awgn := dsp.NewAWGN(-81.0)
	//samp := []int16{64, -28, 1, 34, -73}
	//for i, want := range samp {
	//	got := awgn.Get()
	//	if want != got {
	//		t.Errorf("sample #%d: wanted %d but got %d", i, want, got)
	//	}
	//}

	f, err := os.OpenFile("m81.pcm", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0755)
	defer f.Close()
	if err != nil {
		t.Fatal(err)
	}

	b := make([]byte, 2)
	for i := 0; i < 24000; i++ {
		v := awgn.Get()
		binary.BigEndian.PutUint16(b, *(*uint16)(unsafe.Pointer(&v)))
		_, err := f.Write(b)
		if err != nil {
			t.Fatal(err)
		}
	}

	//for i := 0; i < 50; i++ {
	//	println(awgn.Get())
	//}
}
