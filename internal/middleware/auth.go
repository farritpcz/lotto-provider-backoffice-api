// Package middleware — JWT auth for provider-backoffice-api (#9)
//
// มี 2 audience แยกกัน:
//  - Admin    (frontend #10 = backoffice-admin-web)    — secret: ADMIN_JWT_SECRET
//  - Operator (frontend #11 = backoffice-operator-web) — secret: OPERATOR_JWT_SECRET
//
// ใช้ secret แยกกัน → token ข้าม audience ไม่ได้ (defense-in-depth)
// Pattern ตาม lotto-standalone-admin-api/internal/middleware/auth.go
package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// =============================================================================
// Admin (#10)
// =============================================================================

// AdminClaims ข้อมูลที่เก็บใน admin JWT token
type AdminClaims struct {
	AdminID  int64  `json:"admin_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// AdminJWTAuth middleware ตรวจ admin JWT token (cookie "admin_token" ก่อน, fallback Authorization Bearer)
func AdminJWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractToken(c, "admin_token")
		if tokenString == "" {
			abort401(c, "missing authentication token")
			return
		}

		claims := &AdminClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			abort401(c, "invalid or expired token")
			return
		}

		c.Set("admin_id", claims.AdminID)
		c.Set("admin_username", claims.Username)
		c.Set("admin_role", claims.Role)
		c.Next()
	}
}

// GenerateAdminToken สร้าง JWT token สำหรับ admin
func GenerateAdminToken(adminID int64, username, role, secret string, expiryHours int) (string, error) {
	claims := &AdminClaims{
		AdminID:  adminID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiryHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "lotto-provider-backoffice-api",
			Subject:   "admin",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GetAdminID ดึง admin_id จาก context (0 ถ้าไม่มี)
// AIDEV-NOTE: รองรับทั้ง int64 (ตอน Set ใน middleware) และ float64 (จาก claim ดิบถ้าอ่าน map แทน struct)
func GetAdminID(c *gin.Context) int64 {
	return getInt64FromContext(c, "admin_id")
}

// =============================================================================
// Operator (#11)
// =============================================================================

// OperatorClaims ข้อมูลที่เก็บใน operator JWT token
type OperatorClaims struct {
	OperatorID int64  `json:"operator_id"`
	Username   string `json:"username"`
	jwt.RegisteredClaims
}

// OperatorJWTAuth middleware ตรวจ operator JWT token (cookie "operator_token" ก่อน, fallback Authorization Bearer)
func OperatorJWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractToken(c, "operator_token")
		if tokenString == "" {
			abort401(c, "missing authentication token")
			return
		}

		claims := &OperatorClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			abort401(c, "invalid or expired token")
			return
		}

		c.Set("operator_id", claims.OperatorID)
		c.Set("operator_username", claims.Username)
		c.Next()
	}
}

// GenerateOperatorToken สร้าง JWT token สำหรับ operator
func GenerateOperatorToken(operatorID int64, username, secret string, expiryHours int) (string, error) {
	claims := &OperatorClaims{
		OperatorID: operatorID,
		Username:   username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiryHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "lotto-provider-backoffice-api",
			Subject:   "operator",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GetOperatorID ดึง operator_id จาก context (0 ถ้าไม่มี)
func GetOperatorID(c *gin.Context) int64 {
	return getInt64FromContext(c, "operator_id")
}

// =============================================================================
// Internal helpers
// =============================================================================

// extractToken — อ่าน token จาก cookie ก่อน (httpOnly), ไม่มีค่อย fallback Authorization Bearer
func extractToken(c *gin.Context, cookieName string) string {
	if cookie, err := c.Cookie(cookieName); err == nil && cookie != "" {
		return cookie
	}
	if authHeader := c.GetHeader("Authorization"); authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
			return parts[1]
		}
	}
	return ""
}

func abort401(c *gin.Context, msg string) {
	c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": msg})
	c.Abort()
}

func getInt64FromContext(c *gin.Context, key string) int64 {
	v, ok := c.Get(key)
	if !ok {
		return 0
	}
	if id, ok := v.(int64); ok {
		return id
	}
	if idF, ok := v.(float64); ok {
		return int64(idF)
	}
	return 0
}
