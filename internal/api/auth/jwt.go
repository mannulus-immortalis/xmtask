package auth

import (
	"encoding/base64"

	"github.com/golang-jwt/jwt/v5"

	"github.com/mannulus-immortalis/xmtask/internal/models"
)

type auth struct {
	key []byte
}

type identity struct {
	jwt.RegisteredClaims
	Roles []string `json:"roles"`
}

func New(key string) (*auth, error) {
	data, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return nil, err
	}
	return &auth{
		key: data,
	}, nil
}

func (a *auth) Generate(roles []string) (string, error) {
	t := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"roles": roles,
		},
	)
	return t.SignedString(a.key)
}

func (a *auth) TokenHasRole(tokenString, role string) (bool, error) {
	var i identity
	_, err := jwt.ParseWithClaims(tokenString, &i, a.keyfunc())
	if err != nil {
		return false, err
	}
	for _, r := range i.Roles {
		if r == role {
			return true, nil
		}
	}
	return false, nil
}

func (a *auth) keyfunc() jwt.Keyfunc {
	return func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, models.ErrJWTInvalidMethod
		}
		return a.key, nil
	}
}
