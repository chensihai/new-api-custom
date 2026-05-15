package setting

var (
	WechatPayEnabled    bool   = false
	WechatPayMchID      string = ""
	WechatPayAPIv3Key   string = ""
	WechatPaySerialNo   string = ""
	WechatPayPrivateKey string = ""
	WechatPayMinTopUp   int    = 1
)

var OnWechatPayConfigChange func()
