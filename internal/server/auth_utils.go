package server

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/superles/yapgofermart/internal/model"
)

func (s *Server) GetAuthToken(user model.User) (string, error) {
	expirationTime := jwt.TimeFunc().Add(jswTokenDuration) // Время жизни токена
	claims := &JWTClaims{
		UserID:   user.ID,
		Username: user.Name,
		Role:     user.Role,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(s.cfg.SecretKeyBytes)
	if err != nil {
		return "", err

	}
	return signedToken, nil
}
