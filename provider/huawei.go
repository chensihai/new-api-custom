package provider

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"
)

type HuaweiPhoneAuthProvider struct {
	appId      string
	appSecret  string
	signName   string
	templateId string
	sender     string
	enabled    bool
}

func init() {
	RegisterProvider(&HuaweiPhoneAuthProvider{})
}

func (p *HuaweiPhoneAuthProvider) GetName() string {
	return "Huawei"
}

func (p *HuaweiPhoneAuthProvider) GetProviderID() string {
	return "huawei"
}

func (p *HuaweiPhoneAuthProvider) IsEnabled() bool {
	p.loadConfig()
	return p.enabled
}

func (p *HuaweiPhoneAuthProvider) loadConfig() {
	settings := system_setting.GetPhoneAuthSettings()
	p.appId = settings.HuaweiAppId
	p.appSecret = settings.HuaweiAppSecret
	p.signName = settings.HuaweiSignName
	p.templateId = settings.HuaweiTemplateId
	p.sender = settings.HuaweiSender
	p.enabled = settings.HuaweiEnabled
}

func (p *HuaweiPhoneAuthProvider) ExchangePhoneByToken(ctx context.Context, token string) (string, error) {
	p.loadConfig()
	if !p.enabled {
		return "", fmt.Errorf("huawei phone auth provider is not enabled")
	}
	if !p.ValidateCredentials() {
		return "", fmt.Errorf("huawei credentials not configured")
	}
	return exchangePhoneByHuaweiToken(p.appId, p.appSecret, token)
}

func (p *HuaweiPhoneAuthProvider) SendVerifyCode(ctx context.Context, phone string, code string) error {
	p.loadConfig()
	if !p.enabled {
		return fmt.Errorf("huawei phone auth provider is not enabled")
	}
	if !p.ValidateCredentials() {
		return fmt.Errorf("huawei credentials not configured")
	}
	return sendHuaweiSms(p.appId, p.appSecret, p.signName, p.templateId, p.sender, phone, code)
}

func (p *HuaweiPhoneAuthProvider) ValidateCredentials() bool {
	p.loadConfig()
	return p.appId != "" && p.appSecret != ""
}

func huaweiHmacSha256(key, data []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

func huaweiSha256Hex(data string) string {
	h := sha256.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func huaweiGetAccessToken(appId, appSecret string) (string, error) {
	payload, _ := common.Marshal(map[string]string{
		"grant_type":    "client_credentials",
		"client_id":     appId,
		"client_secret": appSecret,
	})

	req, err := http.NewRequest("POST", "https://iam.myhuaweicloud.com/v3/tokens", strings.NewReader(string(payload)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := common.GetGlobalHttpClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		Token struct {
			ID string `json:"id"`
		} `json:"token"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := common.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if result.Error.Message != "" {
		return "", fmt.Errorf("%s", result.Error.Message)
	}

	return result.Token.ID, nil
}

func exchangePhoneByHuaweiToken(appId, appSecret, token string) (string, error) {
	accessToken, err := huaweiGetAccessToken(appId, appSecret)
	if err != nil {
		common.SysError(fmt.Sprintf("huawei get access token failed: %v", err))
		return "", fmt.Errorf("手机号认证服务鉴权失败")
	}

	payload, _ := common.Marshal(map[string]string{
		"token": token,
	})

	req, err := http.NewRequest("POST", "https://phoneauth.myhuaweicloud.com/v1/phoneauth/getphone", strings.NewReader(string(payload)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", accessToken)

	resp, err := common.GetGlobalHttpClient().Do(req)
	if err != nil {
		common.SysError(fmt.Sprintf("huawei GetPhone request failed: %v", err))
		return "", fmt.Errorf("手机号认证服务调用失败")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		PhoneNumber string `json:"phone_number"`
		ErrorMsg    string `json:"error_msg"`
		ErrorCode   string `json:"error_code"`
	}
	if err := common.Unmarshal(body, &result); err != nil {
		common.SysError(fmt.Sprintf("huawei GetPhone parse failed: %v, body: %s", err, string(body)))
		return "", fmt.Errorf("手机号认证服务响应解析失败")
	}

	if result.ErrorCode != "" {
		common.SysError(fmt.Sprintf("huawei GetPhone error: code=%s, message=%s", result.ErrorCode, result.ErrorMsg))
		return "", fmt.Errorf("手机号认证失败")
	}

	return result.PhoneNumber, nil
}

func sendHuaweiSms(appId, appSecret, signName, templateId, sender, phone, code string) error {
	now := time.Now()
	date := now.UTC().Format("20060102T150405Z")

	payload, _ := common.Marshal(map[string]any{
		"from":          sender,
		"to":            phone,
		"templateId":    templateId,
		"templateParas": []string{code},
		"signature":     signName,
	})

	req, err := http.NewRequest("POST", fmt.Sprintf("https://smsapi.myhuaweicloud.com/v2/%s/sms/messages", appId), strings.NewReader(string(payload)))
	if err != nil {
		return err
	}

	canonicalRequest := fmt.Sprintf("POST\n/v2/%s/sms/messages\n\ncontent-type:application/json; charset=utf-8\nhost:smsapi.myhuaweicloud.com\nx-sdk-date:%s\n\ncontent-type;host;x-sdk-date\n%s",
		appId, date, huaweiSha256Hex(string(payload)))

	credentialScope := fmt.Sprintf("%s/cn-north-1/sms/sdk_request", date[:8])
	stringToSign := fmt.Sprintf("SDK-HMAC-SHA256\n%s\n%s\n%s", date, credentialScope, huaweiSha256Hex(canonicalRequest))

	signature := huaweiHmacSha256([]byte(appSecret), []byte(stringToSign))

	authorization := fmt.Sprintf("SDK-HMAC-SHA256 Access=%s, SignedHeaders=content-type;host;x-sdk-date, Signature=%s", appId, signature)

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Host", "smsapi.myhuaweicloud.com")
	req.Header.Set("X-SDK-Date", date)
	req.Header.Set("Authorization", authorization)

	resp, err := common.GetGlobalHttpClient().Do(req)
	if err != nil {
		common.SysError(fmt.Sprintf("huawei SendSms request failed: %v", err))
		return fmt.Errorf("短信发送服务调用失败")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		ErrorCode string `json:"errorCode"`
		ErrorMsg  string `json:"errorMsg"`
	}
	if err := common.Unmarshal(body, &result); err != nil {
		common.SysError(fmt.Sprintf("huawei SendSms parse failed: %v, body: %s", err, string(body)))
		return fmt.Errorf("短信发送服务响应解析失败")
	}

	if result.Code != "000000" && result.ErrorCode != "" {
		common.SysError(fmt.Sprintf("huawei SendSms error: code=%s, message=%s", result.ErrorCode, result.ErrorMsg))
		return fmt.Errorf("短信发送失败")
	}

	common.SysLog(fmt.Sprintf("huawei SMS sent successfully to %s", phone))
	return nil
}
