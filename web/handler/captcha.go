package handler

import (
	"ispace/common/id"

	"github.com/gin-gonic/gin"
	"github.com/steambap/captcha"
)

type Captcha struct {
	CaptchaIdHeader string // 响应客户端验证码 ID 的 Header 名称
	CaptchaIdQuery  string // 验证码 ID 的请求参数名称
	CaptchaQuery    string // 验证码的请求参数名称
}

func (c *Captcha) ServeHTTP(ctx *gin.Context) error {

	// 生成验证码图片
	image, err := captcha.New(150, 50, func(options *captcha.Options) {
		options.TextLength = 6  // 字符长度
		options.CurveNumber = 4 // 干扰线数量
	})
	if err != nil {
		return err
	}

	// id & 文本
	captchaId := id.UUID()
	//text := image.Text

	// 刷到redis缓存
	//if err := rdb.Client().Set(context.Background(), c.key(captchaId), text, time.Second*30).Err(); err != nil {
	//	return err
	//}

	ctx.Header(c.CaptchaIdHeader, captchaId)
	ctx.Header("Content-Type", "image/png")
	ctx.Header("Cache-Control", "no-cache, no-store")

	//return image.WriteGIF(ctx.Writer, &gif.Options{
	//	NumColors: 256, // NumColors是图像中使用的最大颜色数。它的范围是1到256。
	//	Quantizer: nil, //palette.Plan9被用来代替nil Quantizer，它被用来生成一个具有NumColors大小的调色板。
	//	Drawer:    nil, // draw.FloydSteinberg用于将源图像转换为所需的调色板。Draw.FloydSteinberg用于替代一个无的Drawer。
	//})
	return image.WriteImage(ctx.Writer)
}

func Validate(ctx *gin.Context) error {
	//
	return nil
}

var DefaultCaptcha = &Captcha{
	CaptchaIdHeader: "X-Captcha-Id",
	CaptchaIdQuery:  "captchaId",
	CaptchaQuery:    "captcha",
}
