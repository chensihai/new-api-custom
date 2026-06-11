package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/provider"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
)

func GetPhoneAuthEnabled(c *gin.Context) {
	settings := system_setting.GetPhoneAuthSettings()
	hasProvider := settings.HasAnyProviderEnabled()
	allowPhoneRegister := common.PhoneRegisterEnabled && common.PhoneLoginEnabled && hasProvider

	var configHint string
	if !allowPhoneRegister {
		switch {
		case !common.PhoneRegisterEnabled:
			configHint = "phone_register_not_enabled"
		case !common.PhoneLoginEnabled:
			configHint = "phone_login_not_enabled"
		case !hasProvider:
			configHint = "no_sms_provider_configured"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"enabled":              common.PhoneLoginEnabled && hasProvider,
			"phone_login_enabled":  common.PhoneLoginEnabled,
			"allow_phone_register": allowPhoneRegister,
			"force_real_name_auth": common.PhoneAuthForceRealNameAuth,
			"one_click_available":  common.PhoneLoginEnabled && hasProvider,
			"sms_available":        common.PhoneLoginEnabled && hasProvider,
			"config_hint":          configHint,
		},
	})
}

func SendPhoneVerifyCode(c *gin.Context) {
	phone := c.PostForm("phone")
	if phone == "" {
		var req struct {
			Phone   string `json:"phone"`
			Purpose string `json:"purpose"`
		}
		if err := common.DecodeJson(c.Request.Body, &req); err == nil {
			phone = req.Phone
		}
	}

	if !common.IsValidPhone(phone) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid phone number format",
		})
		return
	}
	phone = common.NormalizePhone(phone)

	clientIP := c.ClientIP()
	if err := service.SendVerifyCode(phone, "login", clientIP); err != nil {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "verification code sent",
	})
}

func PhoneSmsLogin(c *gin.Context) {
	var req struct {
		Phone          string `json:"phone"`
		Code           string `json:"code"`
		TurnstileToken string `json:"turnstile_token"`
	}
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request",
		})
		return
	}

	if !common.IsValidPhone(req.Phone) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid phone number format",
		})
		return
	}
	req.Phone = common.NormalizePhone(req.Phone)

	user, err := service.PhoneLoginByVerifyCode(req.Phone, req.Code)
	if err != nil {
		errMsg := err.Error()
		common.SysLog(fmt.Sprintf("phone sms login failed: phone=%s, error=%s", req.Phone, errMsg))
		status := http.StatusOK
		if strings.Contains(errMsg, "disabled") {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{
			"success": false,
			"message": errMsg,
		})
		return
	}

	setupLogin(user, c)
}

func PhoneRegister(c *gin.Context) {
	var req struct {
		Phone          string `json:"phone"`
		Code           string `json:"code"`
		Username       string `json:"username"`
		Password       string `json:"password"`
		TurnstileToken string `json:"turnstile_token"`
		AffCode        string `json:"aff_code"`
	}
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request",
		})
		return
	}

	if !common.IsValidPhone(req.Phone) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid phone number format",
		})
		return
	}
	req.Phone = common.NormalizePhone(req.Phone)

	user, err := service.PhoneRegister(req.Phone, req.Code, req.Username, req.Password, req.AffCode)
	if err != nil {
		errMsg := err.Error()
		common.SysLog(fmt.Sprintf("phone register failed: phone=%s, username=%s, error=%s", req.Phone, req.Username, errMsg))
		status := http.StatusOK
		if strings.Contains(errMsg, "disabled") {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{
			"success": false,
			"message": errMsg,
		})
		return
	}

	setupLogin(user, c)
}

func PhoneOneClickLogin(c *gin.Context) {
	var req struct {
		Provider string `json:"provider"`
		Token    string `json:"token"`
	}
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request",
		})
		return
	}

	user, err := service.PhoneLoginByToken(req.Provider, req.Token)
	if err != nil {
		errMsg := err.Error()
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": errMsg,
		})
		return
	}

	setupLogin(user, c)
}

func GetPhoneAuthStatus(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "not authenticated",
		})
		return
	}

	status, err := service.GetPhoneAuthStatus(userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    status,
	})
}

func BindPhone(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "not authenticated",
		})
		return
	}

	var req struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request",
		})
		return
	}

	if !common.IsValidPhone(req.Phone) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid phone number format",
		})
		return
	}
	req.Phone = common.NormalizePhone(req.Phone)

	if err := service.BindPhone(userId, req.Phone, req.Code); err != nil {
		errMsg := err.Error()
		common.SysLog(fmt.Sprintf("bind phone failed: userId=%d, phone=%s, error=%s", userId, req.Phone, errMsg))
		status := http.StatusBadRequest
		if strings.Contains(errMsg, "already bound") {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "phone bound successfully",
	})
}

func RebindPhone(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "not authenticated",
		})
		return
	}

	var req struct {
		NewPhone string `json:"new_phone"`
		Code     string `json:"code"`
	}
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request",
		})
		return
	}

	if !common.IsValidPhone(req.NewPhone) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid phone number format",
		})
		return
	}
	req.NewPhone = common.NormalizePhone(req.NewPhone)

	if err := service.RebindPhone(userId, req.NewPhone, req.Code); err != nil {
		errMsg := err.Error()
		common.SysLog(fmt.Sprintf("rebind phone failed: userId=%d, newPhone=%s, error=%s", userId, req.NewPhone, errMsg))
		status := http.StatusBadRequest
		if strings.Contains(errMsg, "already bound") {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{
			"success": false,
			"message": errMsg,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "phone rebound successfully",
	})
}

func UnbindPhone(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "not authenticated",
		})
		return
	}

	var req struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request",
		})
		return
	}

	if req.Phone == "" || req.Code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "phone and verification code are required",
		})
		return
	}

	if !common.IsValidPhone(req.Phone) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid phone number format",
		})
		return
	}
	req.Phone = common.NormalizePhone(req.Phone)

	if err := service.UnbindPhoneWithVerify(userId, req.Phone, req.Code); err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not authenticated") {
			status = http.StatusUnauthorized
		}
		c.JSON(status, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "phone unbound successfully",
	})
}

func GetPhoneAuthProviders(c *gin.Context) {
	settings := system_setting.GetPhoneAuthSettings()
	masked := settings.MaskCredentials()

	allProviders := provider.GetAllProviders()
	providerList := make([]gin.H, 0)
	for _, p := range allProviders {
		providerList = append(providerList, gin.H{
			"id":      p.GetProviderID(),
			"name":    p.GetName(),
			"enabled": p.IsEnabled(),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"settings":  masked,
			"providers": providerList,
		},
	})
}

func UpdatePhoneAuthProviders(c *gin.Context) {
	var req system_setting.PhoneAuthSettings
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request",
		})
		return
	}

	currentSettings := system_setting.GetPhoneAuthSettings()

	if req.AliyunAccessKeySecret == "******" {
		req.AliyunAccessKeySecret = currentSettings.AliyunAccessKeySecret
	}
	if req.TencentSecretKey == "******" {
		req.TencentSecretKey = currentSettings.TencentSecretKey
	}
	if req.HuaweiAppSecret == "******" {
		req.HuaweiAppSecret = currentSettings.HuaweiAppSecret
	}

	*currentSettings = req
	currentSettings.ApplyToCommon()

	configMap, err := config.ConfigToMap(currentSettings)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "failed to serialize settings",
		})
		return
	}

	values := make(map[string]string, len(configMap))
	for k, v := range configMap {
		values["phone_auth."+k] = v
	}

	if err := model.UpdateOptionsBulk(values); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "failed to save settings",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "phone auth settings updated",
	})
}

func AdminGetUserPhone(c *gin.Context) {
	userIdStr := c.Param("id")
	userId, err := strconv.Atoi(userIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid user id",
		})
		return
	}

	user, err := model.GetUserById(userId, true)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "user not found",
		})
		return
	}

	maskedPhone := ""
	if user.PhoneNumber != "" {
		maskedPhone = common.MaskPhone(user.PhoneNumber)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"masked_phone":        maskedPhone,
			"phone_auth_provider": user.PhoneAuthProvider,
		},
	})
}

func AdminUnbindUserPhone(c *gin.Context) {
	userIdStr := c.Param("id")
	userId, err := strconv.Atoi(userIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid user id",
		})
		return
	}

	if err := service.AdminUnbindPhone(userId); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "phone unbound successfully",
	})
}
