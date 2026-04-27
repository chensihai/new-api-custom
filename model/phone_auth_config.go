package model

type PhoneAuthConfig struct {
	Id         int    `json:"id" gorm:"primaryKey"`
	Provider   string `json:"provider" gorm:"type:varchar(32);uniqueIndex"`
	Enabled    bool   `json:"enabled" gorm:"default:false"`
	Credentials string `json:"credentials" gorm:"type:text"`
	CreatedAt  int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

func GetPhoneAuthConfig(provider string) (*PhoneAuthConfig, error) {
	var config PhoneAuthConfig
	err := DB.Where("provider = ?", provider).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func GetAllPhoneAuthConfigs() ([]*PhoneAuthConfig, error) {
	var configs []*PhoneAuthConfig
	err := DB.Find(&configs).Error
	return configs, err
}

func GetEnabledPhoneAuthConfigs() ([]*PhoneAuthConfig, error) {
	var configs []*PhoneAuthConfig
	err := DB.Where("enabled = ?", true).Find(&configs).Error
	return configs, err
}

func UpdatePhoneAuthConfig(config *PhoneAuthConfig) error {
	return DB.Save(config).Error
}

func (c *PhoneAuthConfig) IsEnabled() bool {
	return c.Enabled
}
