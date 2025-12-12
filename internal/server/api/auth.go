package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/BlakeLiAFK/letsync/internal/server/middleware"
	"github.com/BlakeLiAFK/letsync/internal/server/service"
)

// 登录限流配置
const (
	maxLoginAttempts  = 5               // 最大登录尝试次数
	lockoutDuration   = 15 * time.Minute // 锁定时间
	attemptExpiry     = 5 * time.Minute  // 尝试记录过期时间
)

// loginAttempt 登录尝试记录
type loginAttempt struct {
	attempts  int
	firstTime time.Time
	lockUntil time.Time
}

// loginLimiter 登录限流器
type loginLimiter struct {
	mu       sync.RWMutex
	attempts map[string]*loginAttempt
}

var limiter = &loginLimiter{
	attempts: make(map[string]*loginAttempt),
}

// isLocked 检查 IP 是否被锁定
func (l *loginLimiter) isLocked(ip string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if attempt, exists := l.attempts[ip]; exists {
		if time.Now().Before(attempt.lockUntil) {
			return true
		}
	}
	return false
}

// recordAttempt 记录登录尝试
func (l *loginLimiter) recordAttempt(ip string, success bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if success {
		// 登录成功，清除记录
		delete(l.attempts, ip)
		return
	}

	now := time.Now()
	attempt, exists := l.attempts[ip]

	if !exists || now.Sub(attempt.firstTime) > attemptExpiry {
		// 新记录或过期记录
		l.attempts[ip] = &loginAttempt{
			attempts:  1,
			firstTime: now,
		}
		return
	}

	attempt.attempts++

	if attempt.attempts >= maxLoginAttempts {
		// 达到最大尝试次数，锁定账户
		attempt.lockUntil = now.Add(lockoutDuration)
	}
}

// getRemainingLockTime 获取剩余锁定时间
func (l *loginLimiter) getRemainingLockTime(ip string) time.Duration {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if attempt, exists := l.attempts[ip]; exists {
		remaining := time.Until(attempt.lockUntil)
		if remaining > 0 {
			return remaining
		}
	}
	return 0
}

// cleanup 清理过期记录（可定期调用）
func (l *loginLimiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	for ip, attempt := range l.attempts {
		// 如果锁定已过期且尝试记录也过期，则删除
		if now.After(attempt.lockUntil) && now.Sub(attempt.firstTime) > attemptExpiry {
			delete(l.attempts, ip)
		}
	}
}

type AuthHandler struct {
	settings *service.SettingsService
	logger   *service.LogService
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		settings: service.NewSettingsService(),
		logger:   service.NewLogService(),
	}
}

// Login 登录
func (h *AuthHandler) Login(c *gin.Context) {
	clientIP := middleware.GetRealIP(c)

	// 检查是否被锁定
	if limiter.isLocked(clientIP) {
		remaining := limiter.getRemainingLockTime(clientIP)
		h.logger.WarnWithContext(c, "auth", "登录被限制", map[string]interface{}{
			"remaining": remaining.String(),
		})
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error": gin.H{
				"code":    "TOO_MANY_ATTEMPTS",
				"message": "登录尝试次数过多，请稍后再试",
			},
		})
		return
	}

	var req struct {
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "密码不能为空",
			},
		})
		return
	}

	if !h.settings.CheckAdminPassword(req.Password) {
		// 记录失败尝试
		limiter.recordAttempt(clientIP, false)
		h.logger.WarnWithContext(c, "auth", "登录失败", nil)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "密码错误",
			},
		})
		return
	}

	// 登录成功，清除失败记录
	limiter.recordAttempt(clientIP, true)
	h.logger.InfoWithContext(c, "auth", "登录成功", nil)

	token, expiresAt, err := middleware.GenerateToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "生成 Token 失败",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"expires_at": expiresAt,
	})
}

// SetupPassword 首次设置密码
func (h *AuthHandler) SetupPassword(c *gin.Context) {
	if !h.settings.IsFirstRun() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "密码已设置",
			},
		})
		return
	}

	var req struct {
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "密码不能为空",
			},
		})
		return
	}

	// 验证密码强度
	if err := h.settings.ValidatePasswordStrength(req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "WEAK_PASSWORD",
				"message": err.Error(),
			},
		})
		return
	}

	if err := h.settings.SetAdminPassword(req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "设置密码失败",
			},
		})
		return
	}

	h.logger.InfoWithContext(c, "auth", "首次设置密码", nil)

	// 设置完密码后自动登录
	token, expiresAt, err := middleware.GenerateToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "生成 Token 失败",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"expires_at": expiresAt,
	})
}

// ChangePassword 修改密码
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "参数错误",
			},
		})
		return
	}

	if !h.settings.CheckAdminPassword(req.OldPassword) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "原密码错误",
			},
		})
		return
	}

	// 验证新密码强度
	if err := h.settings.ValidatePasswordStrength(req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "WEAK_PASSWORD",
				"message": err.Error(),
			},
		})
		return
	}

	if err := h.settings.SetAdminPassword(req.NewPassword); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "修改密码失败",
			},
		})
		return
	}

	h.logger.InfoWithContext(c, "auth", "密码已修改", nil)

	c.JSON(http.StatusOK, gin.H{
		"message": "密码修改成功",
	})
}

// Status 获取认证状态
func (h *AuthHandler) Status(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"first_run": h.settings.IsFirstRun(),
	})
}
