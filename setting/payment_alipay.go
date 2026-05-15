package setting

var (
	AlipayEnabled    bool   = false
	AlipaySandbox    bool   = false
	AlipayAppId      string = ""
	AlipayPrivateKey string = ""
	AlipayPublicKey  string = ""
	AlipayMinTopUp   int    = 1
)

var OnAlipayConfigChange func()
