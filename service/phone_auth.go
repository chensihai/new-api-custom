package service

import (
	"context"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/provider"
)

func PhoneLoginByToken(providerID string, operatorToken string) (*model.User, error) {
	p := provider.GetProvider(providerID)
	if p == nil {
		return nil, fmt.Errorf("phone auth provider not found: %s", providerID)
	}
	if !p.IsEnabled() {
		return nil, fmt.Errorf("phone auth provider not enabled: %s", providerID)
	}

	phone, err := p.ExchangePhoneByToken(context.Background(), operatorToken)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange phone by token: %w", err)
	}

	if !common.IsValidChinesePhone(phone) {
		return nil, fmt.Errorf("invalid phone number from provider")
	}

	user, err := model.GetUserByPhone(phone)
	if err != nil {
		if !common.PhoneLoginEnabled {
			return nil, fmt.Errorf("phone number not registered")
		}
		user, err = autoRegisterByPhone(phone, providerID)
		if err != nil {
			return nil, fmt.Errorf("failed to auto register: %w", err)
		}
	}

	return user, nil
}

func PhoneLoginByVerifyCode(phone string, code string) (*model.User, error) {
	result := VerifyCode(phone, code)
	if !result.Success {
		return nil, fmt.Errorf(result.Error)
	}

	user, err := model.GetUserByPhone(phone)
	if err != nil {
		if !common.PhoneLoginEnabled {
			return nil, fmt.Errorf("phone number not registered")
		}
		user, err = autoRegisterByPhone(phone, "sms")
		if err != nil {
			return nil, fmt.Errorf("failed to auto register: %w", err)
		}
	}

	return user, nil
}

func PhoneLoginByPassword(phone string, password string) (*model.User, error) {
	user, err := model.GetUserByPhone(phone)
	if err != nil {
		return nil, fmt.Errorf("phone number not registered")
	}

	if user.Status != common.UserStatusEnabled {
		return nil, fmt.Errorf("user is disabled")
	}

	if user.Password == "" {
		return nil, fmt.Errorf("password not set, please use verification code login")
	}

	if !common.ValidatePasswordAndHash(password, user.Password) {
		return nil, fmt.Errorf("incorrect password")
	}

	return user, nil
}

func PhoneRegister(phone string, code string, username string, password string) (*model.User, error) {
	if !common.PhoneLoginEnabled {
		return nil, fmt.Errorf("phone register is disabled")
	}

	if username == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}

	exists, _ := model.CheckUserExistOrDeleted(username, "")
	if exists {
		return nil, fmt.Errorf("username already taken")
	}

	result := VerifyCode(phone, code)
	if !result.Success {
		return nil, fmt.Errorf(result.Error)
	}

	bound, err := model.IsPhoneBound(phone)
	if err != nil {
		return nil, fmt.Errorf("failed to check phone binding: %w", err)
	}
	if bound {
		return nil, fmt.Errorf("phone number already registered")
	}

	user := &model.User{
		Username: username,
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Quota:    common.QuotaForNewUser,
		Group:    "default",
		AffCode:  common.GetRandomString(8),
	}

	if err := user.Insert(0); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if err := user.BindPhone(phone, "sms"); err != nil {
		return nil, fmt.Errorf("failed to bind phone: %w", err)
	}

	if password != "" {
		hashedPassword, err := common.Password2Hash(password)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		user.Password = hashedPassword
		if err := user.Update(false); err != nil {
			return nil, fmt.Errorf("failed to set password: %w", err)
		}
	}

	return user, nil
}

func autoRegisterByPhone(phone string, providerID string) (*model.User, error) {
	for i := 0; i < 3; i++ {
		username := "phone_" + common.GetRandomString(8)

		exists, _ := model.CheckUserExistOrDeleted(username, "")
		if exists {
			continue
		}

		user := &model.User{
			Username: username,
			Role:     common.RoleCommonUser,
			Status:   common.UserStatusEnabled,
			Quota:    common.QuotaForNewUser,
			Group:    "default",
			AffCode:  common.GetRandomString(8),
		}

		if err := user.Insert(0); err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}

		if err := user.BindPhone(phone, providerID); err != nil {
			return nil, fmt.Errorf("failed to bind phone: %w", err)
		}

		return user, nil
	}

	return nil, fmt.Errorf("failed to generate unique username after 3 attempts")
}

func BindPhone(userId int, phone string, code string) error {
	result := VerifyCode(phone, code)
	if !result.Success {
		return fmt.Errorf(result.Error)
	}

	bound, err := model.IsPhoneBound(phone)
	if err != nil {
		return fmt.Errorf("failed to check phone binding: %w", err)
	}
	if bound {
		return fmt.Errorf("phone number already bound to another account")
	}

	user, err := model.GetUserById(userId, true)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	return user.BindPhone(phone, "sms")
}

func RebindPhone(userId int, newPhone string, code string) error {
	result := VerifyCode(newPhone, code)
	if !result.Success {
		return fmt.Errorf(result.Error)
	}

	bound, err := model.IsPhoneBound(newPhone)
	if err != nil {
		return fmt.Errorf("failed to check phone binding: %w", err)
	}
	if bound {
		return fmt.Errorf("phone number already bound to another account")
	}

	user, err := model.GetUserById(userId, true)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	return user.BindPhone(newPhone, "sms")
}

func UnbindPhone(userId int) error {
	user, err := model.GetUserById(userId, true)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	if user.PhoneNumber == "" {
		return fmt.Errorf("no phone number bound to this account")
	}

	hasPassword := user.Password != ""
	hasOAuth := user.GitHubId != "" || user.DiscordId != "" || user.OidcId != "" ||
		user.WeChatId != "" || user.TelegramId != "" || user.LinuxDOId != ""

	if !hasPassword && !hasOAuth {
		return fmt.Errorf("cannot unbind phone: no other login method available")
	}

	return user.UnbindPhone()
}

func GetPhoneAuthStatus(userId int) (map[string]interface{}, error) {
	user, err := model.GetUserById(userId, true)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	status := map[string]interface{}{
		"phone_bound":         user.PhoneNumber != "",
		"masked_phone":        "",
		"phone_auth_verified": user.PhoneAuthVerified,
		"phone_auth_provider": user.PhoneAuthProvider,
	}

	if user.PhoneNumber != "" {
		decrypted, err := common.DecryptPhone(user.PhoneNumber)
		if err == nil {
			status["masked_phone"] = common.MaskPhone(decrypted)
		}
	}

	if user.PhoneAuthTime != nil {
		status["phone_auth_time"] = *user.PhoneAuthTime
	}

	return status, nil
}

func AdminUnbindPhone(userId int) error {
	user, err := model.GetUserById(userId, true)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	if user.PhoneNumber == "" {
		return fmt.Errorf("no phone number bound to this account")
	}

	user.PhoneNumber = ""
	user.PhoneAuthVerified = false
	user.PhoneAuthTime = nil
	user.PhoneAuthProvider = ""

	return user.Update(false)
}
