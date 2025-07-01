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


func ValidateToken(tokenString string) (string, error) {
	tkn, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.NewValidationError("unexpected signing method", jwt.ValidationErrorSignatureInvalid)
		}
		return []byte("secret"), nil
	})
	if err != nil {
		return "", err
	}
	
	if claims, ok := tkn.Claims.(jwt.MapClaims); ok && tkn.Valid {
		if uid, ok := claims["uid"].(string); ok {
			return uid, nil
		}
		return "", jwt.NewValidationError("invalid token claims", jwt.ValidationErrorClaimsInvalid)
	}
	return "", jwt.NewValidationError("invalid token", jwt.ValidationErrorMalformed)
}