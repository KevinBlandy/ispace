package service

import (
	"context"
	"errors"
	"ispace/common"
	"ispace/common/id"
	"ispace/common/response"
	"ispace/rdb"
	"ispace/repo/model"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

type Session struct {
	Id      string // 会话 ID
	Subject int64  // Subject

	// TODO 待优化
	UserAgent  string    // 客户端
	RemoteAddr string    // 客户端 IP
	IssuedAt   time.Time // 签发时间
	ExpireAt   time.Time // 过期时间
	LastAccess time.Time // 最后访问时间
}

type SessionService struct {
	sysConfigService *SysConfigService // 系统配置
	nameSpace        string            // 命名空间
	redisConn        *redis.Client     // Redis
}

func NewAuthService(nameSpace string, redisConn *redis.Client, service *SysConfigService) *SessionService {
	return &SessionService{
		nameSpace:        nameSpace,
		redisConn:        redisConn,
		sysConfigService: service,
	}
}

// sessionKey 构建带命名空间的 session Key，使用 _ 分割项目
func (a *SessionService) sessionKey(parts ...string) string {
	return strings.Join(append([]string{a.nameSpace}, parts...), "::")
}

// Issue 签发 Session
func (a *SessionService) Issue(ctx context.Context, subjectId int64) (string, error) {

	jwtId := id.UUID()                          // Token ID
	subject := strconv.FormatInt(subjectId, 10) // SubjectID
	now := time.Now()                           // 签发时间

	// 签发 Token
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS512, &jwt.RegisteredClaims{
		ID:       jwtId,
		Subject:  subject,
		IssuedAt: jwt.NewNumericDate(now),
	}).SignedString([]byte(a.sysConfigService.Get(ctx, model.SysConfigKeySessionSecret).Value))

	//result, err := auth.JWTEncode(
	//	jwtId,
	//	subject,
	//	a.sysConfigService.Get(model.SysConfigKeySessionSecret).Value,
	//)

	if err != nil {
		return "", err
	}

	// 过期时间
	expire := a.sysConfigService.Get(ctx, model.SysConfigKeySessionExpire).DurationValue()

	// 缓存到 Redis
	_, err = rdb.ExecuteClient(a.redisConn, func(conn *redis.Conn) (any, error) {
		return conn.Set(ctx, a.sessionKey(subject, jwtId), now.Add(expire).UnixMilli(), expire).Result()
	})

	return signed, err
}

// Parse 解析 Session
func (a *SessionService) Parse(ctx context.Context, signed string) (*Session, error) {

	// 解码
	var claims jwt.RegisteredClaims

	token, err := jwt.NewParser().ParseWithClaims(signed, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(a.sysConfigService.Get(ctx, model.SysConfigKeySessionSecret).Value), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, common.NewServiceError(http.StatusUnauthorized, response.Fail(response.CodeUnauthorized).WithMessage("Invalid Token"))
	}

	jwtId := claims.ID
	subject, err := claims.GetSubject()
	if err != nil {
		return nil, err
	}

	// 从缓存获取
	result, err := rdb.ExecuteClient(a.redisConn, func(conn *redis.Conn) (string, error) {
		return conn.Get(ctx, a.sessionKey(subject, jwtId)).Result()
	})

	if err != nil {
		if errors.Is(err, redis.Nil) {
			// 过期
			err = common.NewServiceError(http.StatusUnauthorized, response.Fail(response.CodeUnauthorized).WithMessage("Token Expired"))
		} else {
			slog.ErrorContext(ctx, "查询 session 缓存异常", slog.String("err", err.Error()))
		}
		return nil, err
	}

	subjectId, err := strconv.ParseInt(subject, 10, 64)
	if err != nil {
		return nil, err
	}
	expireAt, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		return nil, err
	}

	return &Session{
		Id:       jwtId,
		Subject:  subjectId,
		ExpireAt: time.UnixMilli(expireAt),
	}, nil
}

// Renewal 续约 Session
func (a *SessionService) Renewal(ctx context.Context, session *Session) (bool, error) {

	expire := a.sysConfigService.Get(ctx, model.SysConfigKeySessionExpire).DurationValue()

	return rdb.ExecuteClient(a.redisConn, func(conn *redis.Conn) (bool, error) {
		return conn.SetXX(ctx,
			a.sessionKey(strconv.FormatInt(session.Subject, 10), session.Id),
			time.Now().Add(expire).UnixMilli(),
			expire,
		).Result()
		//return conn.ExpireXX(ctx,
		//	a.sessionKey(strconv.FormatInt(session.Subject, 10), session.Id),
		//	a.sysConfigService.Get(model.SysConfigKeySessionExpire).DurationValue(),
		//).Result()
	})
}

// Invalid 失效 Session
func (a *SessionService) Invalid(session *Session) error {
	_, err := rdb.ExecuteClient(a.redisConn, func(conn *redis.Conn) (int64, error) {
		return conn.Del(context.Background(), a.sessionKey(strconv.FormatInt(session.Subject, 10), session.Id)).Result()
	})
	return err
}

//// keyFunc 返回 jwt 加密 key
//func (a *SessionService) keyFunc(_ *jwt.Token) (any, error) {
//	return []byte(a.sysConfigService.Get(context.Background(), model.SysConfigKeySessionSecret).Value), nil
//}

// DefaultMemberSessionService 用户会话
var DefaultMemberSessionService = sync.OnceValue(func() *SessionService {
	return NewAuthService("session_member", rdb.Get(), DefaultSysConfigService)
})

// DefaultManagerSessionService 管理员会话
var DefaultManagerSessionService = sync.OnceValue(func() *SessionService {
	return NewAuthService("session_manager", rdb.Get(), DefaultSysConfigService)
})
