package provider

import "context"

type PhoneAuthProvider interface {
	GetName() string
	GetProviderID() string
	IsEnabled() bool
	ExchangePhoneByToken(ctx context.Context, token string) (string, error)
	SendVerifyCode(ctx context.Context, phone string, code string) error
	ValidateCredentials() bool
}

type PhoneAuthResult struct {
	Phone    string
	Provider string
}
