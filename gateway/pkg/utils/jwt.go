package utils

import "github.com/golang-jwt/jwt/v4"

func GenToken(uid string) (string, error) {
	tkn := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid": uid,
	})

	token, err := tkn.SignedString([]byte("secret"))
	if err != nil {
		return "", err
	}

	return token, nil
}
