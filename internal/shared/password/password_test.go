package password

import (
	"strings"
	"testing"
)

func TestHash_ProducesBcrypt(t *testing.T) {
	h, err := Hash("igh2026ok")
	if err != nil {
		t.Fatalf("Hash 失败: %v", err)
	}
	if !strings.HasPrefix(h, "$2") {
		t.Fatalf("哈希不是 bcrypt 格式: %s", h)
	}
	if strings.Contains(h, "igh2026ok") {
		t.Fatal("哈希里出现了明文口令")
	}
}

// 同一个口令两次哈希必须不同（bcrypt 自带随机盐）。
// 如果相同，说明又退回了不加盐的实现。
func TestHash_IsSalted(t *testing.T) {
	a, _ := Hash("igh2026ok")
	b, _ := Hash("igh2026ok")
	if a == b {
		t.Fatal("两次哈希结果相同，说明没有加盐")
	}
	ok1, _ := Verify("igh2026ok", a)
	ok2, _ := Verify("igh2026ok", b)
	if !ok1 || !ok2 {
		t.Fatal("加盐后两个哈希都应能通过校验")
	}
}

func TestVerify_RejectsWrongPassword(t *testing.T) {
	h, _ := Hash("igh2026ok")
	if ok, _ := Verify("igh2026oK", h); ok {
		t.Fatal("错误口令不应通过")
	}
}

// 旧的不加盐 SHA-256 必须继续能登录，并标记为需要升级。
func TestVerify_LegacySHA256(t *testing.T) {
	// sha256("admin123")
	legacy := "240be518fabd2724ddb6f04eeb1da5967448d7e831c08c8fa822809f74c720a9"
	ok, needsUpgrade := Verify("admin123", legacy)
	if !ok {
		t.Fatal("旧哈希应能通过校验，否则现场账号全部登录不了")
	}
	if !needsUpgrade {
		t.Fatal("旧哈希应被标记为需要升级")
	}
	if ok, _ := Verify("wrong", legacy); ok {
		t.Fatal("错误口令不应通过旧哈希校验")
	}
}

func TestVerify_BcryptDoesNotNeedUpgrade(t *testing.T) {
	h, _ := Hash("igh2026ok")
	ok, needsUpgrade := Verify("igh2026ok", h)
	if !ok || needsUpgrade {
		t.Fatalf("bcrypt 哈希应通过且不需要升级，got ok=%v upgrade=%v", ok, needsUpgrade)
	}
}

func TestVerify_EmptyHash(t *testing.T) {
	if ok, _ := Verify("x", ""); ok {
		t.Fatal("空哈希不应通过")
	}
}

func TestValidate_Length(t *testing.T) {
	if err := Validate("1234567"); err == nil {
		t.Fatal("7 位口令应被拒绝")
	}
	if err := Validate("12345678"); err != nil {
		t.Fatalf("8 位口令应被接受: %v", err)
	}
	// 中文按字符数而不是字节数算：6 个汉字不足 8 位，应被拒绝
	if err := Validate("口令六位是吧"); err == nil {
		t.Fatal("6 个汉字应被拒绝（不足 8 位）")
	}
	// 8 个汉字（"八个汉字的口令啊"）刚好达标
	if err := Validate("八个汉字的口令啊"); err != nil {
		t.Fatalf("8 个汉字应被接受: %v", err)
	}
}

// bcrypt 只取前 72 字节，更长的部分会被静默丢弃。
// 与其让人以为超长口令更安全，不如直接拒绝。
func TestValidate_RejectsOver72Bytes(t *testing.T) {
	long := strings.Repeat("a", MaxLength+1)
	if err := Validate(long); err == nil {
		t.Fatal("超过 72 字节的口令应被拒绝")
	}
}

func TestValidate_RejectsSurroundingSpace(t *testing.T) {
	if err := Validate(" igh2026ok"); err == nil {
		t.Fatal("以空格开头的口令应被拒绝（多为复制粘贴带入）")
	}
}

func TestIsLegacy(t *testing.T) {
	h, _ := Hash("igh2026ok")
	if IsLegacy(h) {
		t.Error("bcrypt 不应被判为旧格式")
	}
	if !IsLegacy("240be518fabd2724ddb6f04eeb1da5967448d7e831c08c8fa822809f74c720a9") {
		t.Error("SHA-256 应被判为旧格式")
	}
}
