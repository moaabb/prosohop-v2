package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin" // Import Gin instead of Fiber
	"github.com/moaabb/ecommerce/auth_svc/domain/user"
	"github.com/moaabb/ecommerce/auth_svc/infra/config"
	"github.com/moaabb/ecommerce/auth_svc/infra/database/userdb"
	"github.com/moaabb/ecommerce/auth_svc/utils"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	repository user.Repository
	l          *zap.Logger
	cfg        *config.Config
}

func NewHandler(repo *userdb.Repository, z *zap.Logger, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		repository: repo,
		l:          z,
		cfg:        cfg,
	}
}

func (ah *AuthHandler) Login(c *gin.Context) {
	var u user.User
	err := c.BindJSON(&u)
	if err != nil {
		ah.l.Error("error parsing json", zap.Error(err))
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "error parsing json"})
		return
	}

	ah.l.Info("Fetching User on database")
	user, err := ah.repository.GetUserByEmail(u.Email)
	if err != nil {
		ah.l.Error("error while fetching user", zap.Error(err))
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "could not fetch user"})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(u.Password))
	if err != nil {
		ah.l.Error("error comparing password", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token, _, err := utils.GenerateJWT(user.Id, ah.cfg.JwtSecret)
	if err != nil {
		ah.l.Error("error generating password", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("token", token, 3600, "/", "172.21.193.94", false, true)

	user.Password = ""

	c.JSON(http.StatusOK, user)
}

func (ah *AuthHandler) ValidateRequest(c *gin.Context) {
	token := strings.Split(c.Request.Header.Get("Authorization"), "Bearer ")
	ah.l.Info("validating token...")
	if len(token) < 2 {
		ah.l.Error("token not found in the request")
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	userId, err := utils.VerifyJWT(token[1], ah.cfg.JwtSecret)
	if err != nil {
		ah.l.Error("invalid token", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "invalid token",
		})
		return
	}

	ah.l.Info("request validated")
	c.JSON(http.StatusOK, gin.H{
		"userId": userId,
	})
}
