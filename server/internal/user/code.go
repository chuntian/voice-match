package user

import (
	"crypto/rand"
	"math/big"
)

// generateNumericCode 生成 n 位数字验证码
func generateNumericCode(n int) (string, error) {
	const digits = "0123456789"
	buf := make([]byte, n)
	for i := 0; i < n; i++ {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		buf[i] = digits[idx.Int64()]
	}
	return string(buf), nil
}
