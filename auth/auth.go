package auth

import (
	"errors"
	"strconv"

	"github.com/golang-jwt/jwt/v5"
)

var ErrBadToken = errors.New("bad token")

// JWTEncode 生成 Token
func JWTEncode(id string, subject int64, key []byte) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS512, &jwt.RegisteredClaims{
		//Token Id
		ID: id,
		// 用户 ID
		Subject: strconv.FormatInt(subject, 10),
		// 过期时间
		//ExpiresAt: expiresAt.Unix(),
	}).SignedString(key)
}

// JWTDecode 解析 JWT，会根据key进行校验
func JWTDecode(signedString string, key []byte) (*jwt.RegisteredClaims, error) {
	token, err := jwt.NewParser().ParseWithClaims(signedString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (i interface{}, err error) {
		return key, nil
	})
	if err != nil {
		// 如果Token无效（KEY不对，过期了，还未生效都会返回Error）
		return nil, err
	}
	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		// 返回StandardClaims
		// Token.Valid 验证基于时间的声明，例如：过期时间（ExpiresAt）、签发者（Issuer）、生效时间（Not Before）
		// 需要注意的是，如果没有任何声明在令牌中，仍然会被认为是有效的。
		return claims, nil
	}
	// 非法 Token
	return nil, ErrBadToken
}
