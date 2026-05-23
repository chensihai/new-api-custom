package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/provider"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
)

func GetPhoneAuthEnabled(c *gin.Context) {
	settings := system_setting.GetPhoneAuthSettings()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"enabled":              common.PhoneLoginEnabled && settings.HasAnyProviderEnabled(),
			"allow_phone_register": common.PhoneRegisterEnabled,
			"force_real_name_auth": common.PhoneAuthForceRealNameAuth,
			"one_click_available":  common.PhoneLoginEnabled && settings.HasAnyProviderEnabled(),
			"sms_available":        common.PhoneLoginEnabled && settings.HasAnyProviderEnabled(),
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

	if !common.IsValidChinesePhone(phone) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid phone number format",
		})
		return
	}

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

	user, err := service.PhoneLoginByVerifyCode(req.Phone, req.Code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	setupLogin(user, c)
}

func PhoneRegister(c *gin.Context) {
	var req struct {
		Phone    string `json:"phone"`
		Code     string `json:"code"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request",
		})
		return
	}

	if !common.IsValidChinesePhone(req.Phone) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid phone number format",
		})
		return
	}

	user, err := service.PhoneRegister(req.Phone, req.Code, req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
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
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": err.Error(),
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

	if err := service.BindPhone(userId, req.Phone, req.Code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
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

	if err := service.RebindPhone(userId, req.NewPhone, req.Code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
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

	if err := service.UnbindPhone(userId); err != nil {
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
		decrypted, err := common.DecryptPhone(user.PhoneNumber)
		if err == nil {
			maskedPhone = common.MaskPhone(decrypted)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"masked_phone":         maskedPhone,
			"phone_auth_verified":  user.PhoneAuthVerified,
			"phone_auth_provider":  user.PhoneAuthProvider,
		},
	})
}

func AdminSetUserPhoneVerified(c *gin.Context) {
	userIdStr := c.Param("id")
	userId, err := strconv.Atoi(userIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid user id",
		})
		return
	}

	var req struct {
		Verified bool `json:"verified"`
	}
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request",
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

	if req.Verified {
		now := common.GetTimestamp()
		err = model.DB.Model(user).Updates(map[string]interface{}{
			"phone_auth_verified": true,
			"phone_auth_time":     &now,
		}).Error
	} else {
		err = model.DB.Model(user).Updates(map[string]interface{}{
			"phone_auth_verified": false,
			"phone_auth_time":     nil,
		}).Error
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "failed to update user",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "phone auth verified status updated",
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
