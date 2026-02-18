package filter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"ispace/common"
	"ispace/common/constant"
	"ispace/common/response"
	"ispace/common/types"
	"ispace/db"
	"ispace/repo/model"
	"ispace/web/service"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ShareAuthFilter struct {
	pathName   string // 路径参数名称
	cookieName string // Cookie 名称
	service    *service.ShareService
}

func NewShareAuthFilter(pathName string, cookieName string, service *service.ShareService) *ShareAuthFilter {
	return &ShareAuthFilter{
		pathName:   pathName,
		cookieName: cookieName,
		service:    service,
	}
}

func (s *ShareAuthFilter) Serve(g *gin.Context) (any, error) {

	// 资源路径
	shareId := types.Identifier(g.Param(s.pathName))
	if shareId == "" {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("资源路径不存在"))
	}

	// 查询资源信息
	share, err := db.Transaction(g.Request.Context(), func(ctx context.Context) (*model.Share, error) {
		return s.service.GetByIdentifier(ctx, shareId, "id", "member_id", "password", "path")
	}, db.TxReadOnly)

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if share == nil || share.Id < 1 {
		return nil, nil // 不存在的资源路径，由业务处理
	}

	// 免密
	if share.Password == "" {
		return nil, nil
	}

	// 如果用户已经登陆，且访问资源就是当前用户的分享。则放行
	if g.GetInt64(constant.CtxKeySubject) == share.MemberId {
		return nil, nil
	}

	content, err := g.Cookie(s.cookieName)
	if err != nil {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeSharePasswordFailed).WithMessage("资源需要口令"))
	}

	// "sign-timestamp"
	parts := strings.Split(content, "-")
	if len(parts) != 2 {
		// 非法的 Cookie
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeSharePasswordFailed).WithMessage("资源需要口令"))
	}

	//  sign = shah256(path, timestamp, password)
	hasher := sha256.New()
	hasher.Write([]byte(share.Path))
	hasher.Write([]byte(parts[1]))
	hasher.Write([]byte(share.Password))
	sign := hex.EncodeToString(hasher.Sum(nil))

	if !strings.EqualFold(sign, parts[0]) {
		// 错误的口令
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeSharePasswordFailed).WithMessage("资源需要口令"))
	}

	expireTimestamp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		// 有效期解码异常
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeSharePasswordFailed).WithMessage("资源需要口令"))
	}

	// 检查是否过期
	if time.Now().UnixMilli() > expireTimestamp {
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeSharePasswordFailed).WithMessage("资源需要口令"))
	}

	// OK
	return nil, nil
}
