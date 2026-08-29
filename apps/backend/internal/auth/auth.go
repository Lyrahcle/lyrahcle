package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Role string

const (
	RoleAdmin     Role = "admin"
	RoleOperator  Role = "operator"
	RoleViewer    Role = "viewer"
	TokenIssuer        = "lyrahcle"
	TokenAudience      = "lyrahcle-api"
)

func ValidRole(r string) bool {
	switch Role(r) {
	case RoleAdmin, RoleOperator, RoleViewer:
		return true
	default:
		return false
	}
}

type Claims struct {
	UserID          uuid.UUID `json:"user_id"`
	Role            Role      `json:"role"`
	SessionIssuedAt int64     `json:"session_iat_ns"`
	jwt.RegisteredClaims
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func IssueToken(secret []byte, userID uuid.UUID, role Role, ttl time.Duration) (string, error) {
	token, _, err := IssueTokenWithClaims(secret, userID, role, ttl)
	return token, err
}

func IssueTokenWithClaims(secret []byte, userID uuid.UUID, role Role, ttl time.Duration) (string, *Claims, error) {
	now := time.Now()
	claims := &Claims{
		UserID:          userID,
		Role:            role,
		SessionIssuedAt: now.UnixNano(),
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Issuer:    TokenIssuer,
			Subject:   userID.String(),
			Audience:  jwt.ClaimStrings{TokenAudience},
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(secret)
	if err != nil {
		return "", nil, err
	}
	return signed, claims, nil
}

func ParseToken(secret []byte, tokenString string) (*Claims, error) {
	return parseToken(secret, tokenString)
}

func ParseTokenWithFallback(current, previous []byte, tokenString string) (*Claims, error) {
	claims, err := parseToken(current, tokenString)
	if err == nil || len(previous) == 0 {
		return claims, err
	}
	return parseToken(previous, tokenString)
}

func parseToken(secret []byte, tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(t *jwt.Token) (interface{}, error) {
			if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, errors.New("auth: unexpected signing method")
			}
			return secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(TokenIssuer),
		jwt.WithAudience(TokenAudience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithNotBeforeRequired(),
		jwt.WithLeeway(30*time.Second),
	)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("auth: invalid token")
	}
	if claims.UserID == uuid.Nil || claims.Subject != claims.UserID.String() || !ValidRole(string(claims.Role)) || claims.ID == "" || claims.SessionIssuedAt <= 0 {
		return nil, errors.New("auth: invalid claims")
	}
	return claims, nil
}

func RoleAtLeast(role Role, minimum Role) bool {
	rank := map[Role]int{RoleViewer: 0, RoleOperator: 1, RoleAdmin: 2}
	return rank[role] >= rank[minimum]
}
