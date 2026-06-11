package system_setting

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
)

type PhoneAuthSettings struct {
	Enabled               bool   `json:"enabled"`
	AliyunEnabled         bool   `json:"aliyun_enabled"`
	AliyunAccessKeyId     string `json:"aliyun_access_key_id"`
	AliyunAccessKeySecret string `json:"aliyun_access_key_secret"`
	AliyunSignName        string `json:"aliyun_sign_name"`
	AliyunTemplateCode    string `json:"aliyun_template_code"`
	TencentEnabled        bool   `json:"tencent_enabled"`
	TencentSecretId       string `json:"tencent_secret_id"`
	TencentSecretKey      string `json:"tencent_secret_key"`
	TencentSmsSdkAppId    string `json:"tencent_sms_sdk_app_id"`
	TencentSignName       string `json:"tencent_sign_name"`
	TencentTemplateId     string `json:"tencent_template_id"`
	HuaweiEnabled         bool   `json:"huawei_enabled"`
	HuaweiAppId           string `json:"huawei_app_id"`
	HuaweiAppSecret       string `json:"huawei_app_secret"`
	HuaweiSignName        string `json:"huawei_sign_name"`
	HuaweiTemplateId      string `json:"huawei_template_id"`
	HuaweiSender          string `json:"huawei_sender"`
	SmsCodeExpireSeconds  int    `json:"sms_code_expire_seconds"`
	SmsDailyPhoneLimit    int    `json:"sms_daily_phone_limit"`
	SmsDailyIPLimit       int    `json:"sms_daily_ip_limit"`
	SmsMaxErrorCount      int    `json:"sms_max_error_count"`
}

var defaultPhoneAuthSettings = PhoneAuthSettings{
	Enabled:              false,
	AliyunEnabled:        false,
	TencentEnabled:       false,
	HuaweiEnabled:        false,
	SmsCodeExpireSeconds: 300,
	SmsDailyPhoneLimit:   10,
	SmsDailyIPLimit:      20,
	SmsMaxErrorCount:     5,
}

func init() {
	config.GlobalConfig.Register("phone_auth", &defaultPhoneAuthSettings)
}

func GetPhoneAuthSettings() *PhoneAuthSettings {
	return &defaultPhoneAuthSettings
}

func (s *PhoneAuthSettings) ApplyToCommon() {
	common.PhoneLoginEnabled = s.Enabled
	common.PhoneRegisterEnabled = s.Enabled
}

func (s *PhoneAuthSettings) HasAnyProviderEnabled() bool {
	return s.AliyunEnabled || s.TencentEnabled || s.HuaweiEnabled
}

func (s *PhoneAuthSettings) MaskCredentials() PhoneAuthSettings {
	masked := *s
	if masked.AliyunAccessKeySecret != "" {
		masked.AliyunAccessKeySecret = "******"
	}
	if masked.TencentSecretKey != "" {
		masked.TencentSecretKey = "******"
	}
	if masked.HuaweiAppSecret != "" {
		masked.HuaweiAppSecret = "******"
	}
	return masked
}
