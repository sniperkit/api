package ctxext

import (
	"context"

	"github.com/sniperkit/api/domain"
)

const (
	tokenStringKey = "token"
	accessTokenKey = "domain.AccessToken"
	userKey        = "domain.User"
	userIdKey      = "userId"
	isAuth         = "isAuth"
)

func IsAuth(ctx context.Context) (bool, bool) {
	auth, ok := ctx.Value(isAuth).(bool)
	return auth, ok
}

func TokenString(ctx context.Context) (string, bool) {
	at, ok := ctx.Value(tokenStringKey).(string)
	return at, ok
}

func UserId(ctx context.Context) (string, bool) {
	at, ok := ctx.Value(userIdKey).(string)
	return at, ok
}

func AccessToken(ctx context.Context) (domain.AccessToken, bool) {
	at, ok := ctx.Value(accessTokenKey).(domain.AccessToken)
	return at, ok
}

func User(ctx context.Context) (domain.User, bool) {
	u, ok := ctx.Value(userKey).(domain.User)
	return u, ok
}

func WithTokenString(ctx context.Context, at string) context.Context {
	return context.WithValue(ctx, tokenStringKey, at)
}

func WithAccessToken(ctx context.Context, at domain.AccessToken) context.Context {
	return context.WithValue(ctx, accessTokenKey, at)
}

func WithUser(ctx context.Context, user domain.User) context.Context {
	return context.WithValue(ctx, userKey, user)
}

func WithUserId(ctx context.Context, userId string) context.Context {
	return context.WithValue(ctx, userIdKey, userId)
}

func WithIsAuth(ctx context.Context, auth bool) context.Context {
	return context.WithValue(ctx, isAuth, auth)
}
