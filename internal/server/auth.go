package server

import (
	"encoding/json"
	"errors"
	"fmt"
	errs "github.com/superles/yapgofermart/internal/errors"
	"github.com/superles/yapgofermart/internal/model"
	"github.com/superles/yapgofermart/internal/utils/logger"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/valyala/fasthttp"
)

const (
	jswTokenDuration = 30 * time.Minute
	defaultRole      = "user"
)

// Credentials представляет структуру для аутентификации пользователя
type Credentials struct {
	Username string `json:"login"`
	Password string `json:"password"`
}

// JWTClaims представляет структуру для хранения данных в JWT токене
type JWTClaims struct {
	UserID   int64  `json:"id"`
	Username string `json:"name"`
	Role     string `json:"role"`
	jwt.StandardClaims
}

// LoginHandler обрабатывает запрос на аутентификацию пользователя и создает JWT токен
func (s *Server) registerUserHandler(ctx *fasthttp.RequestCtx) {
	var authUser Credentials
	body := ctx.Request.Body()

	err := json.Unmarshal(body, &authUser)

	if err != nil {
		ctx.Error("ошибка формата отправки", fasthttp.StatusBadRequest)
		return
	}

	user, err := s.storage.GetUserByName(ctx, authUser.Username)

	if len(user.Name) > 0 {
		ctx.Error("пользователь уже существует", fasthttp.StatusConflict)
		return
	}

	if err != nil && !errors.Is(err, errs.ErrNoRows) {
		logger.Log.Errorf("ошибка запроса пользователя %s", err.Error())
		ctx.Error("ошибка сервера", fasthttp.StatusConflict)
		return
	}

	if len(authUser.Username) == 0 || len(authUser.Password) == 0 {
		ctx.Error("ошибка формата отправки", fasthttp.StatusBadRequest)
		return
	}

	password, err := HashPasswordWithRandomSalt(authUser.Password)
	if err != nil {
		logger.Log.Errorf("ошибка хеша пароля пользователя %s", err.Error())
		ctx.Error("ошибка сервера", fasthttp.StatusInternalServerError)
		return
	}
	user = model.User{Name: authUser.Username, PasswordHash: password, Role: defaultRole}
	var regUser model.User
	regUser, err = s.storage.RegisterUser(ctx, user)

	if err != nil {
		logger.Log.Errorf("ошибка регистрации пользователя %s", err.Error())
		ctx.Error("ошибка регистрации пользователя", fasthttp.StatusInternalServerError)
		return
	}

	authToken, err := s.GetAuthToken(regUser)
	if err != nil {
		logger.Log.Errorf("ошибка генерации токена: %s", err.Error())
		ctx.Error("Failed to generate token", fasthttp.StatusInternalServerError)
		return
	}

	ctx.Response.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))
	ctx.SetStatusCode(fasthttp.StatusOK)
}

// LoginHandler обрабатывает запрос на аутентификацию пользователя и создает JWT токен
func (s *Server) loginUserHandler(ctx *fasthttp.RequestCtx) {
	var authUser Credentials
	body := ctx.Request.Body()

	err := json.Unmarshal(body, &authUser)

	if err != nil {
		logger.Log.Errorf("ошибка формата логина: %s", err.Error())
		ctx.Error("ошибка формата отправки", fasthttp.StatusBadRequest)
		return
	}

	if len(authUser.Username) == 0 || len(authUser.Password) == 0 {
		ctx.Error("ошибка формата отправки", fasthttp.StatusBadRequest)
		return
	}

	var user model.User
	user, err = s.storage.GetUserByName(ctx, authUser.Username)
	if err != nil {
		if errors.Is(err, errs.ErrNoRows) {
			logger.Log.Errorf("пользователь не найден %s", err.Error())
			ctx.Error("неверная пара логин/пароль", fasthttp.StatusUnauthorized)
		} else {
			logger.Log.Errorf("ошибка запроса пользователя %s", err.Error())
			ctx.Error("ошибка сервера", fasthttp.StatusInternalServerError)
		}
		return
	}

	if isValid, err := ValidatePassword(user.PasswordHash, authUser.Password); err != nil {
		logger.Log.Errorf("ошибка валидации пароля %s", err.Error())
		ctx.Error("ошибка сервера", fasthttp.StatusInternalServerError)
		return
	} else if !isValid {
		logger.Log.Errorf("неверный пароль")
		ctx.Error("неверная пара логин/пароль", fasthttp.StatusUnauthorized)
		return
	}

	authToken, err := s.GetAuthToken(user)
	if err != nil {
		logger.Log.Errorf("ошибка генерации токена: %s", err.Error())
		ctx.Error("Failed to generate token", fasthttp.StatusInternalServerError)
		return
	}

	ctx.Response.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))
	ctx.SetStatusCode(fasthttp.StatusOK)
}

// AuthMiddleware представляет промежуточное ПО для проверки JWT токена
func (s *Server) authMiddleware(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		authHeader := string(ctx.Request.Header.Peek("Authorization"))
		if authHeader == "" {
			ctx.Error("Authorization token is required", fasthttp.StatusUnauthorized)
			return
		}
		if !strings.Contains(authHeader, "Bearer ") || len(authHeader) <= len("Bearer ") {
			ctx.Error("Authorization token is invalid", fasthttp.StatusUnauthorized)
			return
		}
		tokenString := authHeader[len("Bearer "):]
		token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return s.cfg.SecretKeyBytes, nil
		})

		if err != nil {
			ctx.Error("Invalid token", fasthttp.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(*JWTClaims)
		if !ok || !token.Valid {
			ctx.Error("Invalid token claims", fasthttp.StatusUnauthorized)
			return
		}

		// Передача информации о пользователе и роли в контексте
		ctx.SetUserValue("userID", claims.UserID)
		ctx.SetUserValue("userName", claims.Username)
		ctx.SetUserValue("userRole", claims.Role)

		next(ctx)
	}
}
