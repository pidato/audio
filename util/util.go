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

package util

import (
	"encoding/hex"
	"math/rand"
	"net"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

func BytesToPointer(b []byte) unsafe.Pointer {
	h := *(*reflect.SliceHeader)(unsafe.Pointer(&b))
	return unsafe.Pointer(h.Data)
}

func I16SliceToPointer(b []int16) unsafe.Pointer {
	h := *(*reflect.SliceHeader)(unsafe.Pointer(&b))
	return unsafe.Pointer(h.Data)
}

// IsRefused Return true if error is ICMP connection refused.
func IsRefused(err error) bool {
	operr, ok := err.(*net.OpError)
	return ok && operr.Err == syscall.ECONNREFUSED
}

// IsUseOfClosed Return true if error was caused by reading from a closed socket.
func IsUseOfClosed(err error) bool {
	return strings.Contains(err.Error(), "use of closed network connection")
}

// IsIPv6 Return true if IP contains a colon.
func IsIPv6(ip string) bool {
	return strings.Index(ip, ":") >= 0
}

// GenerateCSeq Generate a secure random number between 0 and 50,000.
func GenerateCSeq() int {
	return rand.Int() % 50000
}

// GenerateTag Generate a 48-bit secure random string like: 27c97271d363.
func GenerateTag() string {
	return hex.EncodeToString(randomBytes(6))
}

// GenerateBranch Generate a SIP 2.0 Via branch ID. This is probably not suitable for use by
// stateless proxies.
func GenerateBranch() string {
	return "z9hG4bK-" + GenerateTag()
}

// GenerateCallID Generate a secure UUID4, e.g.f47ac10b-58cc-4372-a567-0e02b2c3d479
func GenerateCallID() string {
	lol := randomBytes(15)
	digs := hex.EncodeToString(lol)
	uuid4 := digs[0:8] +
		"-" + digs[8:12] +
		"-4" + digs[12:15] +
		"-a" + digs[15:18] +
		"-" + digs[18:]
	return uuid4
}

// GenerateOriginID Generate a random ID for an SDP.
func GenerateOriginID() string {
	return strconv.FormatUint(uint64(rand.Uint32()), 10)
}

func randomBytes(l int) (b []byte) {
	b = make([]byte, l)
	for i := 0; i < l; i++ {
		b[i] = byte(rand.Int())
	}
	return
}

//func append(buf []byte, s string) []byte {
//	lenb, lens := len(buf), len(s)
//	if lenb+lens <= cap(buf) {
//		buf = buf[0 : lenb+lens]
//	} else {
//		panic("mtu exceeded D:")
//	}
//	for i := 0; i < lens; i++ {
//		buf[lenb+i] = byte(s[i])
//	}
//	return buf
//}

func Portstr(port uint16) string {
	return strconv.FormatInt(int64(port), 10)
}

// StringToBytes converts a string to a byte slice.
//
// This is a shallow copy, means that the returned byte slice reuse
// the underlying array in string, so you can't change the returned
// byte slice in any situations.
func StringToBytes(s string) (b []byte) {
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh.Data = sh.Data
	bh.Len = sh.Len
	bh.Cap = sh.Len
	return b
}

// BytesToString converts a byte slice to a string.
//
// This is a shallow copy, means that the returned string reuse the
// underlying array in byte slice, it's your responsibility to keep
// the input byte slice survive until you don't access the string anymore.
func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func CastInt16ToBytes(pcm []int16) []byte {
	h := *(*reflect.SliceHeader)(unsafe.Pointer(&pcm))
	return *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Data: h.Data,
		Len:  h.Len * 2,
		Cap:  h.Cap * 2,
	}))
}

func CastBytesToInt16(b []byte) []int16 {
	if len(b) < 2 {
		return nil
	}
	if len(b)%2 != 0 {
		b = b[0 : len(b)-1]
	}
	h := *(*reflect.SliceHeader)(unsafe.Pointer(&b))
	return *(*[]int16)(unsafe.Pointer(&reflect.SliceHeader{
		Data: h.Data,
		Len:  h.Len / 2,
		Cap:  h.Cap / 2,
	}))
}

// Sets the bit at pos in the integer n.
func SetBit64(n uint64, pos uint64) uint64 {
	n |= (1 << pos)
	return n
}

// Clears the bit at pos in n.
func ClearBit64(n uint64, pos uint64) uint64 {
	mask := ^(uint64(1) << pos)
	n &= mask
	return n
}

func HasBit64(n uint64, pos uint64) bool {
	val := n & (1 << pos)
	return (val > 0)
}

// Sets the bit at pos in the integer n.
func SetBit16(n uint16, pos uint16) uint16 {
	n |= (1 << pos)
	return n
}

// Clears the bit at pos in n.
func ClearBit16(n uint16, pos uint16) uint16 {
	mask := ^(uint16(1) << pos)
	n &= mask
	return n
}

func HasBit16(n uint16, pos uint16) bool {
	val := n & (1 << pos)
	return (val > 0)
}

// Sets the bit at pos in the integer n.
func SetBit32(n uint32, pos uint32) uint32 {
	n |= (1 << pos)
	return n
}

// Clears the bit at pos in n.
func ClearBit32(n uint32, pos uint32) uint32 {
	mask := ^(uint32(1) << pos)
	n &= mask
	return n
}

func HasBit32(n uint32, pos uint32) bool {
	val := n & (1 << pos)
	return (val > 0)
}
