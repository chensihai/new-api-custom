package provider

import "sync"

var (
	providers     = make(map[string]PhoneAuthProvider)
	providersMu   sync.RWMutex
)

func RegisterProvider(p PhoneAuthProvider) {
	providersMu.Lock()
	defer providersMu.Unlock()
	providers[p.GetProviderID()] = p
}

func GetProvider(id string) PhoneAuthProvider {
	providersMu.RLock()
	defer providersMu.RUnlock()
	return providers[id]
}

func GetEnabledProviders() []PhoneAuthProvider {
	providersMu.RLock()
	defer providersMu.RUnlock()
	var result []PhoneAuthProvider
	for _, p := range providers {
		if p.IsEnabled() {
			result = append(result, p)
		}
	}
	return result
}

func GetAllProviders() []PhoneAuthProvider {
	providersMu.RLock()
	defer providersMu.RUnlock()
	var result []PhoneAuthProvider
	for _, p := range providers {
		result = append(result, p)
	}
	return result
}

func HasEnabledProvider() bool {
	return len(GetEnabledProviders()) > 0
}
