package handler

import (
	"context"
	"errors"
	"ispace/common"
	"ispace/common/constant"
	"ispace/common/id"
	"ispace/common/response"
	"ispace/rdb"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/steambap/captcha"
)

type Captcha struct {
	CaptchaIdHeader string        // 响应客户端验证码 ID 的 Header 名称
	CaptchaIdQuery  string        // 验证码 ID 的请求参数名称
	CaptchaQuery    string        // 验证码的请求参数名称
	CaptchaKey      string        // 验证码缓存 key
	redisClient     *redis.Client // Redis 客户端
}

func (c *Captcha) Serve(ctx *gin.Context) {

	// 生成验证码图片
	image, err := captcha.New(150, 50, func(options *captcha.Options) {
		options.TextLength = 6  // 字符长度
		options.CurveNumber = 4 // 干扰线数量
	})
	if err != nil {

		slog.ErrorContext(context.WithValue(ctx.Request.Context(), constant.CtxKeyLoggerName, "captcha"),
			"验证码生成异常",
			slog.String("err", err.Error()),
		)

		http.Error(ctx.Writer, err.Error(), http.StatusInternalServerError)
		return
	}

	captchaId := id.UUID()

	// 缓存 1 分钟
	_, err = rdb.ExecuteClient(c.redisClient, func(conn *redis.Conn) (string, error) {
		return conn.Set(context.Background(), c.CaptchaKey+captchaId, image.Text, time.Minute).Result()
	})
	if err != nil {
		slog.ErrorContext(context.WithValue(ctx.Request.Context(), constant.CtxKeyLoggerName, "captcha"),
			"验证码缓存异常",
			slog.String("err", err.Error()),
		)
		http.Error(ctx.Writer, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx.Header(c.CaptchaIdHeader, captchaId)
	ctx.Header("Content-Type", "image/png")
	ctx.Header("Cache-Control", "no-cache, no-store")

	if err := image.WriteImage(ctx.Writer); err != nil {
		slog.ErrorContext(
			context.WithValue(ctx.Request.Context(), constant.CtxKeyLoggerName, "captcha"),
			"验证码响应异常",
			slog.String("err", err.Error()),
		)
	}
}

func (c *Captcha) Validate(ctx *gin.Context) (any, error) {

	captchaId := ctx.Query(c.CaptchaIdQuery)
	captchaText := ctx.Query(c.CaptchaQuery)

	if captchaId == "" || captchaText == "" {
		// 验证码为空
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeCaptchaFailed).WithMessage("验证码为空"))
	}
	text, err := rdb.ExecuteClient(c.redisClient, func(conn *redis.Conn) (string, error) {
		return conn.GetDel(context.Background(), c.CaptchaKey+captchaId).Result()
	})
	if err != nil {
		// 验证码过期
		if errors.Is(err, redis.Nil) {
			err = common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeCaptchaFailed).WithMessage("验证码错误"))
		}
		return nil, err
	}
	if !strings.EqualFold(text, captchaText) {
		// 验证码错误
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeCaptchaFailed).WithMessage("验证码错误"))
	}
	return nil, nil
}

var DefaultCaptcha = sync.OnceValue(func() *Captcha {
	return &Captcha{
		CaptchaIdHeader: constant.HttpHeaderCaptchaId,
		CaptchaIdQuery:  "captchaId",
		CaptchaQuery:    "captcha",
		CaptchaKey:      "captcha_",
		redisClient:     rdb.Get(),
	}
})
