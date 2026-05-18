package setting

import (
	"strings"
	"sync"
)

var (
	RebateEnabled        bool
	SettlementPeriod     int
	RebateVisibleGroups  string
	rebateSettingMu      sync.RWMutex
)

const (
	DefaultSettlementPeriod = 7
)

func init() {
	RebateEnabled = false
	SettlementPeriod = DefaultSettlementPeriod
	RebateVisibleGroups = ""
}

func IsRebateEnabled() bool {
	rebateSettingMu.RLock()
	defer rebateSettingMu.RUnlock()
	return RebateEnabled
}

func GetSettlementPeriod() int {
	rebateSettingMu.RLock()
	defer rebateSettingMu.RUnlock()
	return SettlementPeriod
}

func GetRebateVisibleGroups() string {
	rebateSettingMu.RLock()
	defer rebateSettingMu.RUnlock()
	return RebateVisibleGroups
}

func IsUserRebateVisible(userGroup string) bool {
	rebateSettingMu.RLock()
	groups := RebateVisibleGroups
	rebateSettingMu.RUnlock()

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

func UpdateRebateSettings(enabled bool, period int, groups string) {
	rebateSettingMu.Lock()
	defer rebateSettingMu.Unlock()
	RebateEnabled = enabled
	if period >= 1 {
		SettlementPeriod = period
	}
	RebateVisibleGroups = groups
}
