package server

import (
	"context"
	"github.com/superles/yapgofermart/internal/model"
	"github.com/superles/yapgofermart/internal/storage"
	"github.com/valyala/fasthttp"
	"testing"
)

func createRequestWithBody(body string) *fasthttp.RequestCtx {
	reqCtx := fasthttp.RequestCtx{
		Request: fasthttp.Request{},
	}
	reqCtx.Request.SetBodyString(body)
	return &reqCtx
}

func createRequestWithBodyAndContentType(body string, contentType string) *fasthttp.RequestCtx {
	reqCtx := fasthttp.RequestCtx{
		Request: fasthttp.Request{},
	}
	reqCtx.Request.Header.SetContentType(contentType)
	reqCtx.Request.SetBodyString(body)
	return &reqCtx
}

func authCtxWithUser(ctx *fasthttp.RequestCtx, user model.User) {
	ctx.SetUserValue("userID", user.ID)
	ctx.SetUserValue("userName", user.Name)
	ctx.SetUserValue("userRole", user.Role)
}

func generateTestUsers(t *testing.T, storage storage.Storage) []model.User {

	salt, err := GenerateSalt()

	if err != nil {
		t.Fatalf("ошибка инициализации соли: %s", err.Error())
	}

	users := make([]model.User, 2)

	var regUser model.User

	regUser, err = storage.RegisterUser(context.Background(), model.User{
		Name:         "user",
		PasswordHash: HashPassword("pass", salt),
	})

	if err != nil {
		t.Fatalf("ошибка инициализации пользователя: %s", err.Error())
	}

	users[0] = regUser

	regUser, err = storage.RegisterUser(context.Background(), model.User{
		Name:         "user1",
		PasswordHash: HashPassword("pass1", salt),
	})

	if err != nil {
		t.Fatalf("ошибка инициализации пользователя: %s", err.Error())
	}

	users[1] = regUser

	return users
}
