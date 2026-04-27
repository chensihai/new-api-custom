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

type TencentPhoneAuthProvider struct {
	secretId    string
	secretKey   string
	smsSdkAppId string
	signName    string
	templateId  string
	enabled     bool
}

func init() {
	RegisterProvider(&TencentPhoneAuthProvider{})
}

func (p *TencentPhoneAuthProvider) GetName() string {
	return "Tencent"
}

func (p *TencentPhoneAuthProvider) GetProviderID() string {
	return "tencent"
}

func (p *TencentPhoneAuthProvider) IsEnabled() bool {
	p.loadConfig()
	return p.enabled
}

func (p *TencentPhoneAuthProvider) loadConfig() {
	settings := system_setting.GetPhoneAuthSettings()
	p.secretId = settings.TencentSecretId
	p.secretKey = settings.TencentSecretKey
	p.smsSdkAppId = settings.TencentSmsSdkAppId
	p.signName = settings.TencentSignName
	p.templateId = settings.TencentTemplateId
	p.enabled = settings.TencentEnabled
}

func (p *TencentPhoneAuthProvider) ExchangePhoneByToken(ctx context.Context, token string) (string, error) {
	p.loadConfig()
	if !p.enabled {
		return "", fmt.Errorf("tencent phone auth provider is not enabled")
	}
	if !p.ValidateCredentials() {
		return "", fmt.Errorf("tencent credentials not configured")
	}
	return exchangePhoneByTencentToken(p.secretId, p.secretKey, token)
}

func (p *TencentPhoneAuthProvider) SendVerifyCode(ctx context.Context, phone string, code string) error {
	p.loadConfig()
	if !p.enabled {
		return fmt.Errorf("tencent phone auth provider is not enabled")
	}
	if !p.ValidateCredentials() {
		return fmt.Errorf("tencent credentials not configured")
	}
	return sendTencentSms(p.secretId, p.secretKey, p.smsSdkAppId, p.signName, p.templateId, phone, code)
}

func (p *TencentPhoneAuthProvider) ValidateCredentials() bool {
	p.loadConfig()
	return p.secretId != "" && p.secretKey != "" && p.smsSdkAppId != ""
}

func tencentHmacSha256(key, data string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

func tencentSha256Hex(data string) string {
	h := sha256.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func tencentTC3Sign(secretId, secretKey, service, action, host, payload string) (map[string]string, error) {
	algorithm := "TC3-HMAC-SHA256"
	timestamp := time.Now().Unix()
	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, service)

	canonicalRequest := fmt.Sprintf("POST\n/\n\ncontent-type:application/json; charset=utf-8\nhost:%s\n\ncontent-type;host\n%s", host, tencentSha256Hex(payload))

	stringToSign := fmt.Sprintf("%s\n%d\n%s\n%s", algorithm, timestamp, credentialScope, tencentSha256Hex(canonicalRequest))

	secretDate := tencentHmacSha256(fmt.Sprintf("TC3%s", secretKey), date)
	secretService := tencentHmacSha256(secretDate, service)
	secretSigning := tencentHmacSha256(secretService, "tc3_request")
	signature := tencentHmacSha256(secretSigning, stringToSign)

	authorization := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=content-type;host, Signature=%s", algorithm, secretId, credentialScope, signature)

	headers := map[string]string{
		"Content-Type":  "application/json; charset=utf-8",
		"Host":          host,
		"X-TC-Action":   action,
		"X-TC-Timestamp": fmt.Sprintf("%d", timestamp),
		"X-TC-Version":  "2021-01-11",
		"Authorization": authorization,
	}
	return headers, nil
}

func tencentRequest(host, action, payload string, secretId, secretKey, service string) ([]byte, error) {
	headers, err := tencentTC3Sign(secretId, secretKey, service, action, host, payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("https://%s", host), strings.NewReader(payload))
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := common.GetGlobalHttpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func exchangePhoneByTencentToken(secretId, secretKey, token string) (string, error) {
	payload, _ := common.Marshal(map[string]string{
		"Token": token,
	})

	body, err := tencentRequest("portal.tencentcloudapi.com", "GetMobile", string(payload), secretId, secretKey, "portal")
	if err != nil {
		common.SysError(fmt.Sprintf("tencent GetMobile request failed: %v", err))
		return "", fmt.Errorf("手机号认证服务调用失败")
	}

	var result struct {
		Response struct {
			Mobile string `json:"Mobile"`
			Error  struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
		} `json:"Response"`
	}
	if err := common.Unmarshal(body, &result); err != nil {
		common.SysError(fmt.Sprintf("tencent GetMobile parse failed: %v, body: %s", err, string(body)))
		return "", fmt.Errorf("手机号认证服务响应解析失败")
	}

	if result.Response.Error.Code != "" {
		common.SysError(fmt.Sprintf("tencent GetMobile error: code=%s, message=%s", result.Response.Error.Code, result.Response.Error.Message))
		return "", fmt.Errorf("手机号认证失败")
	}

	return result.Response.Mobile, nil
}

func sendTencentSms(secretId, secretKey, smsSdkAppId, signName, templateId, phone, code string) error {
	templateParam, _ := common.Marshal(map[string]string{"code": code})

	payload, _ := common.Marshal(map[string]any{
		"SmsSdkAppId":   smsSdkAppId,
		"SignName":      signName,
		"TemplateId":    templateId,
		"TemplateParam": string(templateParam),
		"PhoneNumberSet": []string{phone},
	})

	body, err := tencentRequest("sms.tencentcloudapi.com", "SendSms", string(payload), secretId, secretKey, "sms")
	if err != nil {
		common.SysError(fmt.Sprintf("tencent SendSms request failed: %v", err))
		return fmt.Errorf("短信发送服务调用失败")
	}

	var result struct {
		Response struct {
			SendStatusSet []struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"SendStatusSet"`
			Error struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
		} `json:"Response"`
	}
	if err := common.Unmarshal(body, &result); err != nil {
		common.SysError(fmt.Sprintf("tencent SendSms parse failed: %v, body: %s", err, string(body)))
		return fmt.Errorf("短信发送服务响应解析失败")
	}

	if result.Response.Error.Code != "" {
		common.SysError(fmt.Sprintf("tencent SendSms error: code=%s, message=%s", result.Response.Error.Code, result.Response.Error.Message))
		return fmt.Errorf("短信发送失败")
	}

	if len(result.Response.SendStatusSet) > 0 && result.Response.SendStatusSet[0].Code != "Ok" {
		common.SysError(fmt.Sprintf("tencent SendSms status error: code=%s, message=%s", result.Response.SendStatusSet[0].Code, result.Response.SendStatusSet[0].Message))
		return fmt.Errorf("短信发送失败")
	}

	common.SysLog(fmt.Sprintf("tencent SMS sent successfully to %s", phone))
	return nil
}
