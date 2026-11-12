// Package password 提供口令哈希与校验。
//
// 历史原因：早期实现用**不加盐的 SHA-256**（见 database.seed），
// 同样的口令在任何系统里都会算出同一个值，一旦库泄露可以直接查表还原。
// 这里改成 bcrypt，并保留对旧哈希的校验，登录成功时透明升级。
package password

import (
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

const (
	// MinLength 是最短口令长度。
	// 工位账号习惯用 6 位数字，但那是历史遗留；新设口令至少 8 位。
	MinLength = 8
	// MaxLengthMySQL 不是限制，而是防御：bcrypt 只取前 72 字节，
	// 更长的口令会被静默截断，反而让人以为更安全。直接拒绝更好。
	MaxLength = 72
)

// Hash 生成 bcrypt 哈希。
func Hash(plain string) (string, error) {
	if err := Validate(plain); err != nil {
		return "", err
	}
	return HashForSeed(plain)
}

// HashForSeed 只生成哈希、不校验强度。
//
// 专供种子数据使用：内置账号的口令（admin123 / 123456）低于新口令的
// 最低长度要求，用 Hash 会导致首次部署直接失败。除此之外不应调用它——
// 用户自设口令一律走 Hash。
func HashForSeed(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("生成口令哈希失败: %w", err)
	}
	return string(b), nil
}

// Validate 校验口令强度。
//
// 只做长度与空白检查，不强制大小写数字符号的组合：
// 现场工人用的是工位机键盘，强制复杂度会导致"写在便利贴上"这种更糟的结果。
func Validate(plain string) error {
	n := utf8.RuneCountInString(plain)
	if n < MinLength {
		return fmt.Errorf("口令至少 %d 位", MinLength)
	}
	if len(plain) > MaxLength {
		return fmt.Errorf("口令不能超过 %d 字节", MaxLength)
	}
	if strings.TrimSpace(plain) != plain {
		return fmt.Errorf("口令不能以空格开头或结尾")
	}
	return nil
}

// Verify 校验明文与存储的哈希是否匹配。
//
// 同时支持两种格式：
//   - bcrypt（$2a$/$2b$/$2y$ 开头）
//   - 旧的 SHA-256 十六进制（64 字符）
//
// 返回 needsUpgrade=true 表示这条是旧哈希，调用方应当在验证成功后
// 用 Hash 重新生成并写回，避免旧哈希长期留在库里。
func Verify(plain, stored string) (ok bool, needsUpgrade bool) {
	if stored == "" {
		return false, false
	}
	if strings.HasPrefix(stored, "$2a$") || strings.HasPrefix(stored, "$2b$") ||
		strings.HasPrefix(stored, "$2y$") {
		return bcrypt.CompareHashAndPassword([]byte(stored), []byte(plain)) == nil, false
	}

	sum := sha256.Sum256([]byte(plain))
	legacy := fmt.Sprintf("%x", sum)
	// 常量时间比较：避免通过响应时间逐字节猜哈希
	if subtle.ConstantTimeCompare([]byte(legacy), []byte(stored)) == 1 {
		return true, true
	}
	return false, false
}

// IsLegacy 判断存储的哈希是否为旧格式（用于启动时统计与告警）。
func IsLegacy(stored string) bool {
	return stored != "" &&
		!strings.HasPrefix(stored, "$2a$") &&
		!strings.HasPrefix(stored, "$2b$") &&
		!strings.HasPrefix(stored, "$2y$")
}
