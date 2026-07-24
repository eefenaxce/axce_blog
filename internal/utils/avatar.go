package utils

import (
	"crypto/md5"
	"encoding/hex"
	"strings"
)

// GenerateAvatarURL returns a default avatar URL based on email.
// QQ email → QQ avatar, others → weavatar.com
func GenerateAvatarURL(email string) string {
	if isQQEmail(email) {
		qqNum := strings.TrimSuffix(email, "@qq.com")
		return "https://q1.qlogo.cn/g?b=qq&nk=" + qqNum + "&s=640"
	}
	hash := md5.Sum([]byte(strings.ToLower(strings.TrimSpace(email))))
	return "https://weavatar.com/avatar/" + hex.EncodeToString(hash[:]) + "?d=identicon&s=200"
}

func isQQEmail(email string) bool {
	return strings.HasSuffix(strings.ToLower(strings.TrimSpace(email)), "@qq.com")
}
