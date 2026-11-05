// Package security 提供后台依赖的口令散列与访问令牌能力。
// 全部基于 Go 标准库实现，避免引入额外模块。
package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	pwdPrefix   = "pbkdf2_sha256"
	pwdIter     = 12000
	pwdKeyLen   = 32
	pwdSaltLen  = 16
	tokenIssuer = "jjxt"
)

// ---------- 口令 ----------

// HashPassword 生成带随机盐的口令散列，格式：pbkdf2_sha256$迭代次数$盐$散列
func HashPassword(plain string) (string, error) {
	if len(plain) < 6 {
		return "", errors.New("密码长度不能少于 6 位")
	}
	salt := make([]byte, pwdSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := pbkdf2SHA256([]byte(plain), salt, pwdIter, pwdKeyLen)
	return fmt.Sprintf("%s$%d$%s$%s", pwdPrefix, pwdIter,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key)), nil
}

// VerifyPassword 校验口令是否与散列匹配
func VerifyPassword(hash, plain string) bool {
	parts := strings.Split(hash, "$")
	if len(parts) != 4 || parts[0] != pwdPrefix {
		return false
	}
	iter, err := strconv.Atoi(parts[1])
	if err != nil || iter <= 0 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}
	got := pbkdf2SHA256([]byte(plain), salt, iter, len(want))
	return subtle.ConstantTimeCompare(got, want) == 1
}

func pbkdf2SHA256(password, salt []byte, iter, keyLen int) []byte {
	prf := hmac.New(sha256.New, password)
	hashLen := prf.Size()
	blocks := (keyLen + hashLen - 1) / hashLen
	out := make([]byte, 0, blocks*hashLen)
	u := make([]byte, hashLen)
	var idx [4]byte
	for block := 1; block <= blocks; block++ {
		prf.Reset()
		prf.Write(salt)
		idx[0] = byte(block >> 24)
		idx[1] = byte(block >> 16)
		idx[2] = byte(block >> 8)
		idx[3] = byte(block)
		prf.Write(idx[:])
		out = prf.Sum(out)
		t := out[len(out)-hashLen:]
		copy(u, t)
		for n := 2; n <= iter; n++ {
			prf.Reset()
			prf.Write(u)
			u = u[:0]
			u = prf.Sum(u)
			for i := range u {
				t[i] ^= u[i]
			}
		}
	}
	return out[:keyLen]
}

// ---------- 访问令牌 ----------

var tokenSecret []byte

// SetTokenSecret 由进程启动时注入签名密钥
func SetTokenSecret(secret string) {
	if secret == "" {
		secret = "jjxt-default-secret-change-me"
	}
	tokenSecret = []byte(secret)
}

// Claims 令牌载荷
type Claims struct {
	Uid      int64  `json:"uid"`
	Username string `json:"usr"`
	Role     string `json:"role"`
	Issuer   string `json:"iss"`
	IssuedAt int64  `json:"iat"`
	ExpireAt int64  `json:"exp"`
}

// SignToken 签发访问令牌，payload 使用 base64url(JSON) + "." + base64url(HMAC)
func SignToken(c Claims, ttl time.Duration) (string, error) {
	if len(tokenSecret) == 0 {
		return "", errors.New("令牌签名密钥未初始化")
	}
	now := time.Now()
	c.Issuer = tokenIssuer
	if c.IssuedAt == 0 {
		c.IssuedAt = now.Unix()
	}
	if c.ExpireAt == 0 {
		c.ExpireAt = now.Add(ttl).Unix()
	}
	body, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(body)
	sig := sign(payload)
	return payload + "." + sig, nil
}

// ParseToken 解析并校验访问令牌
func ParseToken(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, errors.New("令牌格式不正确")
	}
	if subtle.ConstantTimeCompare([]byte(sign(parts[0])), []byte(parts[1])) != 1 {
		return nil, errors.New("令牌签名校验失败")
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, errors.New("令牌内容损坏")
	}
	var c Claims
	if err := json.Unmarshal(body, &c); err != nil {
		return nil, errors.New("令牌内容损坏")
	}
	if c.Issuer != tokenIssuer {
		return nil, errors.New("令牌来源不匹配")
	}
	if c.ExpireAt > 0 && time.Now().Unix() > c.ExpireAt {
		return nil, errors.New("登录已过期，请重新登录")
	}
	return &c, nil
}

func sign(payload string) string {
	mac := hmac.New(sha256.New, tokenSecret)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
