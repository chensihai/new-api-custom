package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/provider"
	"github.com/QuantumNous/new-api/setting/system_setting"
)

const (
	phoneVerifyCodePrefix   = "phone_auth:verify:"
	phoneRateLimitPrefix    = "phone_auth:rate:"
	phoneDailyLimitPrefix   = "phone_auth:daily:"
	phoneIPDailyLimitPrefix = "phone_auth:ip:daily:"
	phoneErrorCountPrefix   = "phone_auth:errors:"
)

var (
	memoryVerifyCache     sync.Map
	memoryRateLimitCache  sync.Map
	memoryDailyLimitCache sync.Map
)

type memoryCacheItem struct {
	Value     string
	ExpiresAt time.Time
}

func GenerateVerifyCode() string {
	max := big.NewInt(1000000)
	n, _ := rand.Int(rand.Reader, max)
	return fmt.Sprintf("%06d", n)
}

func SendVerifyCode(phone string, purpose string, clientIP string) error {
	settings := system_setting.GetPhoneAuthSettings()

	if !common.IsValidPhone(phone) {
		return fmt.Errorf("invalid phone number format")
	}
	phone = common.NormalizePhone(phone)

	phoneHash := common.HashPhone(phone)

	rateKey := phoneRateLimitPrefix + phoneHash
	if err := checkRateLimit(rateKey, 60*time.Second); err != nil {
		return fmt.Errorf("verification code sent too frequently, please try again in 60 seconds")
	}

	today := time.Now().Format("2006-01-02")
	dailyKey := phoneDailyLimitPrefix + phoneHash + ":" + today
	dailyLimit := settings.SmsDailyPhoneLimit
	if dailyLimit <= 0 {
		dailyLimit = 10
	}
	if err := checkDailyLimit(dailyKey, dailyLimit, 24*time.Hour); err != nil {
		return fmt.Errorf("daily SMS limit exceeded for this phone number")
	}

	if clientIP != "" {
		ipDailyKey := phoneIPDailyLimitPrefix + clientIP + ":" + today
		ipDailyLimit := settings.SmsDailyIPLimit
		if ipDailyLimit <= 0 {
			ipDailyLimit = 20
		}
		if err := checkDailyLimit(ipDailyKey, ipDailyLimit, 24*time.Hour); err != nil {
			return fmt.Errorf("daily SMS limit exceeded for this IP")
		}
	}

	code := GenerateVerifyCode()

	enabledProviders := provider.GetEnabledProviders()
	if len(enabledProviders) == 0 {
		return fmt.Errorf("no SMS provider enabled")
	}

	sendErr := enabledProviders[0].SendVerifyCode(context.Background(), phone, code)
	if sendErr != nil {
		common.SysLog("failed to send verify code via " + enabledProviders[0].GetProviderID() + ": " + sendErr.Error())
		return fmt.Errorf("failed to send verification code")
	}

	expireSeconds := settings.SmsCodeExpireSeconds
	if expireSeconds <= 0 {
		expireSeconds = 300
	}

	codeKey := phoneVerifyCodePrefix + phoneHash
	if err := setCache(codeKey, code, time.Duration(expireSeconds)*time.Second); err != nil {
		return fmt.Errorf("failed to store verification code")
	}

	if err := setCache(rateKey, "1", 60*time.Second); err != nil {
		common.SysLog("failed to set rate limit cache: " + err.Error())
	}

	return nil
}

type VerifyCodeResult struct {
	Success bool
	Error   string
}

func VerifyCode(phone string, code string) VerifyCodeResult {
	phoneHash := common.HashPhone(phone)

	settings := system_setting.GetPhoneAuthSettings()
	maxErrors := settings.SmsMaxErrorCount
	if maxErrors <= 0 {
		maxErrors = 5
	}

	codeKey := phoneVerifyCodePrefix + phoneHash
	storedCode, err := getCache(codeKey)
	if err != nil || storedCode == "" {
		return VerifyCodeResult{Success: false, Error: "verification code expired or not found"}
	}

	if storedCode != code {
		errorKey := phoneErrorCountPrefix + phoneHash
		errorCountStr, _ := getCache(errorKey)
		errorCount := 0
		if errorCountStr != "" {
			errorCount, _ = strconv.Atoi(errorCountStr)
		}
		errorCount++

		if errorCount >= maxErrors {
			deleteCache(codeKey)
			deleteCache(errorKey)
			return VerifyCodeResult{Success: false, Error: "too many failed attempts, verification code invalidated"}
		}

		setCache(errorKey, strconv.Itoa(errorCount), 300*time.Second)
		return VerifyCodeResult{Success: false, Error: "incorrect verification code"}
	}

	deleteCache(codeKey)
	deleteCache(phoneErrorCountPrefix + phoneHash)
	return VerifyCodeResult{Success: true}
}

func checkRateLimit(key string, window time.Duration) error {
	val, err := getCache(key)
	if err == nil && val != "" {
		return fmt.Errorf("rate limit exceeded")
	}
	return nil
}

func checkDailyLimit(key string, limit int, window time.Duration) error {
	val, _ := getCache(key)
	count := 0
	if val != "" {
		count, _ = strconv.Atoi(val)
	}
	if count >= limit {
		return fmt.Errorf("daily limit exceeded")
	}
	count++
	setCache(key, strconv.Itoa(count), window)
	return nil
}

func setCache(key, value string, expiration time.Duration) error {
	if common.RedisEnabled {
		return common.RedisSet(key, value, expiration)
	}
	memoryVerifyCache.Store(key, &memoryCacheItem{Value: value, ExpiresAt: time.Now().Add(expiration)})
	return nil
}

func getCache(key string) (string, error) {
	if common.RedisEnabled {
		return common.RedisGet(key)
	}
	if item, ok := memoryVerifyCache.Load(key); ok {
		cached := item.(*memoryCacheItem)
		if time.Now().Before(cached.ExpiresAt) {
			return cached.Value, nil
		}
		memoryVerifyCache.Delete(key)
	}
	return "", fmt.Errorf("not found")
}

func deleteCache(key string) {
	if common.RedisEnabled {
		common.RedisDel(key)
		return
	}
	memoryVerifyCache.Delete(key)
}

func init() {
	go func() {
		for {
			time.Sleep(60 * time.Second)
			now := time.Now()
			memoryVerifyCache.Range(func(key, value interface{}) bool {
				cached := value.(*memoryCacheItem)
				if now.After(cached.ExpiresAt) {
					memoryVerifyCache.Delete(key)
				}
				return true
			})
		}
	}()
}
