package utils

import (
	random "crypto/rand"

	"gitlab.com/go-extension/rand"
)

func GetRand(min, max int) int {
	if min == max {
		return min
	}

	if min > max {
		return max + rand.Standard.IntN(min-max+1)
	}

	return min + rand.Standard.IntN(max-min+1)
}

func GetString(len int) string {
	chars := "0123456789abcdefghigklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	str := make([]byte, len)
	for i := 0; i < len; i++ {
		str[i] = chars[rand.Standard.IntN(62)]
	}

	return string(str)
}

func RandBytes(b []byte) {
	random.Read(b)
}
