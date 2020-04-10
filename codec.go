package main

import (
	"github.com/dgrijalva/jwt-go"
)

type Codec struct {
	Secret string
}

func (c *Codec) createToken(payload map[string]interface{}) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(payload))
	tokenString, err := token.SignedString([]byte(c.Secret))
	if err != nil {
		return "", nil
	}

	return tokenString, nil
}

func (c *Codec) parseToken(jwtToken string) (map[string]interface{}, error) {
	token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) {
		return []byte(c.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	if token.Valid {
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			return claims, nil
		}

		return nil, nil
	}

	return nil, err
}
