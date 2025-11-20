package services

import (
	"errors"
	"fmt"

	"github.com/dgrijalva/jwt-go"
)

func (s *Service) Validate(tokenStr string) (string, error) {
	s.log.Debug("Validating token.........")
	jwtSecret := []byte(s.cfg.Auth.AccessSecKey)

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		s.log.Error("Token parse error: ", err)
		return "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		id, ok := claims["sub"].(string)
		if !ok {
			return "", errors.New("ID not found or not a string in token claims")
		}
		s.log.Debug("Token valid. ID:", id)
		return id, nil
	}
	return "", errors.New("invalid token")
}
