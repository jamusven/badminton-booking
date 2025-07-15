package misc

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"strconv"
)

func Sha1(str string) string {
	h := sha1.New()
	h.Write([]byte(str))
	bs := h.Sum(nil)
	return fmt.Sprintf("%x", bs)
}

func ToINT(str string) int {
	v, _ := strconv.Atoi(str)
	return v
}

func ToINT64(str string) int64 {
	v, _ := strconv.Atoi(str)
	return int64(v)
}

func ToIntFromUint(i uint) int {
	return int(i)
}

func ToFloat32(str string) float32 {
	v, _ := strconv.ParseFloat(str, 32)
	return float32(v)
}

func ToString(i int) string {
	return strconv.Itoa(i)
}

func ToJson(data interface{}) string {
	b, _ := json.Marshal(data)

	return string(b)
}

func ToJsonPrettify(data interface{}) string {
	b, _ := json.MarshalIndent(data, "", "  ")

	return string(b)
}
