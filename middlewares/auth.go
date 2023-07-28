package middlewares

import (
	"errors"
	"fmt"
	"rest-gateway/conf"
	"rest-gateway/logger"
	"rest-gateway/models"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
)

var configuration = conf.GetConfig()
var noNeedAuthRoutes = []string{
	"/",
	"/monitoring/status",
	"/auth/authenticate",
	"/auth/refreshtoken",
	"/monitoring/getresourcesutilization",
}

func isAuthNeeded(path string) bool {
	for _, route := range noNeedAuthRoutes {
		if route == path {
			return false
		}
	}

	return true
}

func extractToken(authHeader string) (string, error) {
	if authHeader == "" {
		return "", errors.New("unsupported auth header")
	}

	splited := strings.Split(authHeader, " ")
	if len(splited) != 2 {
		return "", errors.New("unsupported auth header")
	}

	tokenString := splited[1]
	return tokenString, nil
}

func verifyToken(tokenString string, secret string) (models.AuthSchema, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return models.AuthSchema{}, errors.New("f")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok && !token.Valid {
		return models.AuthSchema{}, errors.New("f")
	}

	var user models.AuthSchema
	if !configuration.USER_PASS_BASED_AUTH {
		user = models.AuthSchema{
			Username:        claims["username"].(string),
			ConnectionToken: claims["connection_token"].(string),
		}
	} else {
		user = models.AuthSchema{
			Username:  claims["username"].(string),
			Password:  claims["password"].(string),
			AccountId: claims["account_id"].(float64),
		}
	}
	return user, nil
}

func Authenticate(c *fiber.Ctx) error {
	log := logger.GetLogger(c)
	path := strings.ToLower(string(c.Context().URI().RequestURI()))
	var user models.AuthSchema
	var err error
	if isAuthNeeded(path) {
		headers := c.GetReqHeaders()
		tokenString, err := extractToken(headers["Authorization"])
		if err != nil || tokenString == "" {
			tokenString = c.Query("authorization")
			if tokenString == "" { // fallback - get the token from the query params
				log.Warnf("Authentication error - jwt token is missing")
				if configuration.DEBUG {
					fmt.Printf("Method: %s, Path: %s, IP: %s\nBody: %s\n", c.Method(), c.Path(), c.IP(), string(c.Body()))
				}
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"message": "Unauthorized",
				})
			}
		}
		user, err = verifyToken(tokenString, configuration.JWT_SECRET)
		if err != nil {
			log.Warnf("Authentication error - jwt token validation has failed")
			if configuration.DEBUG {
				fmt.Printf("Method: %s, Path: %s, IP: %s\nBody: %s\n", c.Method(), c.Path(), c.IP(), string(c.Body()))
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "Unauthorized",
			})
		}
	} else if path == "/auth/refreshtoken" {
		var body models.RefreshTokenSchema
		if err := c.BodyParser(&body); err != nil {
			log.Errorf("Authenticate: %s", err.Error())
			if configuration.DEBUG {
				fmt.Printf("Method: %s, Path: %s, IP: %s\nBody: %s\n", c.Method(), c.Path(), c.IP(), string(c.Body()))
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Server error",
			})
		}

		if body.JwtRefreshToken == "" {
			log.Warnf("Authentication error - refresh token is missing")
			if configuration.DEBUG {
				fmt.Printf("Method: %s, Path: %s, IP: %s\nBody: %s\n", c.Method(), c.Path(), c.IP(), string(c.Body()))
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "Unauthorized",
			})
		}

		user, err = verifyToken(body.JwtRefreshToken, configuration.REFRESH_JWT_SECRET)
		if err != nil {
			log.Warnf("Authentication error - refresh token validation has failed")
			if configuration.DEBUG {
				fmt.Printf("Method: %s, Path: %s, IP: %s\nBody: %s\n", c.Method(), c.Path(), c.IP(), string(c.Body()))
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "Unauthorized",
			})
		}
	}

	c.Locals("userData", user)
	return c.Next()
}
