package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/Nishad4140/api_gateway/authorize"
	"github.com/graphql-go/graphql"
)

var (
	secret []byte
)

func InitMiddlewareSecret(secretString string) {
	secret = []byte(secretString)
}

func ClientMiddleware(next graphql.FieldResolveFn) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {

		r := p.Context.Value("request").(*http.Request)
		cookie, err := r.Cookie("jwtToken")
		if err != nil {
			return nil, err
		}
		if cookie == nil {
			return nil, fmt.Errorf("not logged in")
		}

		ctx := p.Context

		token := cookie.Value

		auth, err := authorize.ValidateToken(token, secret)
		if err != nil {
			fmt.Println(err.Error())
			return nil, err
		}

		userIDval := auth["userID"].(uint)

		if userIDval < 1 {
			return nil, errors.New("userID is not valid")
		}

		ctx = context.WithValue(ctx, "userID", userIDval)

		p.Context = ctx

		return next(p)
	}
}

func AdminMiddleware(next graphql.FieldResolveFn) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {

		r := p.Context.Value("request").(*http.Request)
		cookie, err := r.Cookie("jwtToken")
		if err != nil {
			return nil, err
		}
		if cookie == nil {
			return nil, fmt.Errorf("not logged in")
		}

		ctx := p.Context

		token := cookie.Value

		auth, err := authorize.ValidateToken(token, secret)
		if err != nil {
			fmt.Println(err.Error())
			return nil, err
		}

		userIDval := auth["userID"].(uint)
		if userIDval < 1 {
			return nil, fmt.Errorf("invalid userID")
		}
		if !auth["isAdmin"].(bool) {
			return nil, fmt.Errorf("not an admin")
		}

		ctx = context.WithValue(ctx, "userID", userIDval)

		p.Context = ctx

		return next(p)
	}
}
