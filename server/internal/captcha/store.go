package captcha

import (
	"time"

	"github.com/mojocn/base64Captcha"
)

// driverDigit 使用数字验证码（6 位），避免字母混淆（l/1, O/0）。
// 图片尺寸 120x40，适合移动端展示。DotCount=80 个干扰圆点。
var driver = base64Captcha.NewDriverDigit(40, 120, 6, 0.4, 80)

// store 使用 base64Captcha 内置内存 store：collectNum=1024 条触发回收，过期 10 分钟。
// 多实例部署时建议替换为 Redis store（base64Captcha 支持）。
var store = base64Captcha.NewMemoryStore(1024, 600*time.Second) // 600 秒 = 10 分钟

// captchaInstance 复用 driver + store，提供 Generate / Verify 接口。
var captchaInstance = base64Captcha.NewCaptcha(driver, store)

// generateCaptcha 生成一个新验证码，返回 (id, base64Image, error)。
// imageBase64 已包含 "data:image/png;base64," 前缀，可直接用于 <img src=>。
func generateCaptcha() (id string, imageBase64 string, err error) {
	id, b64s, _, err := captchaInstance.Generate()
	if err != nil {
		return "", "", err
	}
	return id, b64s, nil
}

// verifyCaptcha 校验验证码，校验成功后自动销毁（一次性）。
// 严格大小写不敏感，去空格。
func verifyCaptcha(id, code string) bool {
	if id == "" || code == "" {
		return false
	}
	return store.Verify(id, code, true)
}
