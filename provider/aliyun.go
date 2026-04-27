package provider

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"
)

type AliyunPhoneAuthProvider struct {
	accessKeyId     string
	accessKeySecret string
	signName        string
	templateCode    string
	enabled         bool
}

func init() {
	RegisterProvider(&AliyunPhoneAuthProvider{})
}

func (p *AliyunPhoneAuthProvider) GetName() string {
	return "Aliyun"
}

func (p *AliyunPhoneAuthProvider) GetProviderID() string {
	return "aliyun"
}

func (p *AliyunPhoneAuthProvider) IsEnabled() bool {
	p.loadConfig()
	return p.enabled
}

func (p *AliyunPhoneAuthProvider) loadConfig() {
	settings := system_setting.GetPhoneAuthSettings()
	p.accessKeyId = settings.AliyunAccessKeyId
	p.accessKeySecret = settings.AliyunAccessKeySecret
	p.signName = settings.AliyunSignName
	p.templateCode = settings.AliyunTemplateCode
	p.enabled = settings.AliyunEnabled
}

func (p *AliyunPhoneAuthProvider) ExchangePhoneByToken(ctx context.Context, token string) (string, error) {
	p.loadConfig()
	if !p.enabled {
		return "", fmt.Errorf("aliyun phone auth provider is not enabled")
	}
	if !p.ValidateCredentials() {
		return "", fmt.Errorf("aliyun credentials not configured")
	}
	return exchangePhoneByAliyunToken(p.accessKeyId, p.accessKeySecret, token)
}

func (p *AliyunPhoneAuthProvider) SendVerifyCode(ctx context.Context, phone string, code string) error {
	p.loadConfig()
	if !p.enabled {
		return fmt.Errorf("aliyun phone auth provider is not enabled")
	}
	if !p.ValidateCredentials() {
		return fmt.Errorf("aliyun credentials not configured")
	}
	return sendAliyunSms(p.accessKeyId, p.accessKeySecret, p.signName, p.templateCode, phone, code)
}

func (p *AliyunPhoneAuthProvider) ValidateCredentials() bool {
	p.loadConfig()
	return p.accessKeyId != "" && p.accessKeySecret != ""
}

func aliyunComputeSignature(params url.Values, accessKeySecret string) string {
	var keys []string
	for k := range params {
		if k == "Signature" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf strings.Builder
	for i, k := range keys {
		buf.WriteString(percentEncode(k))
		buf.WriteByte('=')
		buf.WriteString(percentEncode(params.Get(k)))
		if i < len(keys)-1 {
			buf.WriteByte('&')
		}
	}

	stringToSign := "GET&%2F&" + percentEncode(buf.String())

	mac := hmac.New(sha1.New, []byte(accessKeySecret+"&"))
	mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func percentEncode(s string) string {
	s = url.QueryEscape(s)
	s = strings.ReplaceAll(s, "+", "%20")
	s = strings.ReplaceAll(s, "*", "%2A")
	s = strings.ReplaceAll(s, "%7E", "~")
	return s
}

func aliyunRequest(apiUrl string, params url.Values, accessKeySecret string) ([]byte, error) {
	params.Set("SignatureMethod", "HMAC-SHA1")
	params.Set("SignatureVersion", "1.0")
	params.Set("SignatureNonce", fmt.Sprintf("%d-%s", time.Now().UnixNano(), common.GetRandomString(8)))
	params.Set("Timestamp", time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	params.Set("Format", "JSON")

	params.Set("Signature", aliyunComputeSignature(params, accessKeySecret))

	reqUrl := apiUrl + "?" + params.Encode()
	resp, err := common.GetGlobalHttpClient().Get(reqUrl)
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

func exchangePhoneByAliyunToken(accessKeyId, accessKeySecret, token string) (string, error) {
	params := url.Values{}
	params.Set("AccessKeyId", accessKeyId)
	params.Set("Action", "GetPhone")
	params.Set("Version", "2017-05-25")
	params.Set("ApiVersion", "2.0")
	params.Set("Token", token)

	body, err := aliyunRequest("https://dypnsapi.aliyuncs.com/", params, accessKeySecret)
	if err != nil {
		common.SysError(fmt.Sprintf("aliyun GetPhone request failed: %v", err))
		return "", fmt.Errorf("手机号认证服务调用失败")
	}

	var result struct {
		Code            string `json:"Code"`
		Message         string `json:"Message"`
		GetPhoneResult  struct {
			PhoneNumber string `json:"PhoneNumber"`
		} `json:"GetPhoneResult"`
	}
	if err := common.Unmarshal(body, &result); err != nil {
		common.SysError(fmt.Sprintf("aliyun GetPhone parse failed: %v, body: %s", err, string(body)))
		return "", fmt.Errorf("手机号认证服务响应解析失败")
	}

	if result.Code != "OK" {
		common.SysError(fmt.Sprintf("aliyun GetPhone error: code=%s, message=%s", result.Code, result.Message))
		return "", fmt.Errorf("手机号认证失败")
	}

	return result.GetPhoneResult.PhoneNumber, nil
}

func sendAliyunSms(accessKeyId, accessKeySecret, signName, templateCode, phone, code string) error {
	templateParam, _ := common.Marshal(map[string]string{"code": code})

	params := url.Values{}
	params.Set("AccessKeyId", accessKeyId)
	params.Set("Action", "SendSms")
	params.Set("Version", "2017-05-25")
	params.Set("PhoneNumbers", phone)
	params.Set("SignName", signName)
	params.Set("TemplateCode", templateCode)
	params.Set("TemplateParam", string(templateParam))

	body, err := aliyunRequest("https://dysmsapi.aliyuncs.com/", params, accessKeySecret)
	if err != nil {
		common.SysError(fmt.Sprintf("aliyun SendSms request failed: %v", err))
		return fmt.Errorf("短信发送服务调用失败")
	}

	var result struct {
		Code    string `json:"Code"`
		Message string `json:"Message"`
	}
	if err := common.Unmarshal(body, &result); err != nil {
		common.SysError(fmt.Sprintf("aliyun SendSms parse failed: %v, body: %s", err, string(body)))
		return fmt.Errorf("短信发送服务响应解析失败")
	}

	if result.Code != "OK" {
		common.SysError(fmt.Sprintf("aliyun SendSms error: code=%s, message=%s", result.Code, result.Message))
		return fmt.Errorf("短信发送失败")
	}

	common.SysLog(fmt.Sprintf("aliyun SMS sent successfully to %s", phone))
	return nil
}
