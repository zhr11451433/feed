package account

import (
	"backend/internal/apierror"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AccountHandler struct {
	accountService *AccountService
}

func NewAccountHandler(accountService *AccountService) *AccountHandler {
	return &AccountHandler{accountService: accountService}
}

func (h *AccountHandler) CreateAccount(c *gin.Context) {
	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apierror.JSONBody(err))
		return
	}
	err := h.accountService.CreateAccount(c.Request.Context(), &Account{
		Username: req.Username,
		Password: req.Password,
	})
	if errors.Is(err, ErrUsernameTaken) {
		c.JSON(http.StatusConflict, apierror.JSONBody(err))
		return
	}
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), apierror.JSONBody(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "account created"})
}

func (h *AccountHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apierror.JSONBody(err))
		return
	}
	account, access, refresh, err := h.accountService.Login(c.Request.Context(), req.Username, req.Password)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), apierror.JSONBody(err))
		return
	}
	c.JSON(http.StatusOK, LoginResponse{
		Token:        access,
		RefreshToken: refresh,
		AccountID:    account.ID,
		Username:     account.Username,
	})
}

func (h *AccountHandler) FindByID(c *gin.Context) {
	var req FindByIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apierror.JSONBody(err))
		return
	}
	a, err := h.accountService.FindByID(c.Request.Context(), req.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), apierror.JSONBody(err))
		return
	}
	c.JSON(http.StatusOK, a)
}

func (h *AccountHandler) FindByUsername(c *gin.Context) {
	var req FindByUsernameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apierror.JSONBody(err))
		return
	}
	a, err := h.accountService.FindByUsername(c.Request.Context(), req.Username)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), apierror.JSONBody(err))
		return
	}
	c.JSON(http.StatusOK, a)
}

func (h *AccountHandler) Logout(c *gin.Context) {
	id, err := getAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, apierror.JSONBody(err))
		return
	}
	if err := h.accountService.Logout(c.Request.Context(), id); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), apierror.JSONBody(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "account logged out"})
}

func (h *AccountHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apierror.JSONBody(err))
		return
	}
	access, id, username, err := h.accountService.RefreshAccessToken(c.Request.Context(), req.RefreshToken)
	if err != nil && err.Error() == "invalid refresh token" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), apierror.JSONBody(err))
		return
	}
	c.JSON(http.StatusOK, LoginResponse{Token: access, AccountID: id, Username: username})
}

func getAccountID(c *gin.Context) (uint, error) {
	v, ok := c.Get("accountID")
	if !ok {
		return 0, errors.New("accountID not found")
	}
	id, ok := v.(uint)
	if !ok {
		return 0, errors.New("accountID has invalid type")
	}
	return id, nil
}
