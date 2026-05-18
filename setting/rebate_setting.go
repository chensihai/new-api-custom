package setting

import (
	"strings"

	"github.com/QuantumNous/new-api/setting/config"
)

type RebateSetting struct {
	Enabled        bool   `json:"enabled"`
	SettlementPeriod int  `json:"settlement_period"`
	VisibleGroups  string `json:"visible_groups"`
}

var rebateSetting = RebateSetting{
	Enabled:          false,
	SettlementPeriod: 7,
	VisibleGroups:    "",
}

func init() {
	config.GlobalConfig.Register("rebate_setting", &rebateSetting)
}

func IsRebateEnabled() bool {
	return rebateSetting.Enabled
}

func GetSettlementPeriod() int {
	return rebateSetting.SettlementPeriod
}

func GetRebateVisibleGroups() string {
	return rebateSetting.VisibleGroups
}

func IsUserRebateVisible(userGroup string) bool {
	groups := rebateSetting.VisibleGroups
	if groups == "" {
		return false
	}
	userGroups := strings.Split(userGroup, ",")
	visibleGroups := strings.Split(groups, ",")
	for _, ug := range userGroups {
		ug = strings.TrimSpace(ug)
		if ug == "" {
			continue
		}
		for _, vg := range visibleGroups {
			vg = strings.TrimSpace(vg)
			if ug == vg {
				return true
			}
		}
	}
	return false
}

func GetRebateSetting() *RebateSetting {
	return &rebateSetting
}
