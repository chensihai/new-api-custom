# Changelog

All notable changes to this project will be documented in this file.

## [v1.1.0] - 2026-05-18

### Added
- **邀请返利系统**：被邀请用户充值时邀请人按比例获得返利
  - 五态生命周期：pending → settled → transferred / pending → frozen → clawed_back
  - 四级风控：延迟结算期 → 退款冻结 → 扣回 → 欠扣抵扣
  - 分组可见性权限 + 返利上限封顶
  - 4张新表：rebate_accounts, rebate_records, rebate_deficits, rebate_cap_progresses
  - User扩展字段：rebate_rate（万分比）, rebate_cap
  - 12条API路由（6用户 + 6管理员）
  - 异步返利触发（gopool.Go）+ 定时结算任务（5分钟轮询）
  - 前端web/default：返利概览卡片、划转弹窗、设置Section、用户编辑扩展
  - 前端web/classic：RebateCard、SettingsRebate、EditUserModal扩展
- **支付宝PreCreate预下单**（alipay.trade.precreate）
  - 替代TradePagePay（URL拼接）模式，真正在支付宝服务端创建交易
  - 前端渲染二维码供用户扫码（qrcode.react），15分钟超时
  - 修复沙箱环境下"二维码过期"问题
- **支付宝/微信支付官方SDK集成**
  - smartwalle/alipay v3 集成
  - 充值回调（gopool.Go异步触发返利）
  - web/default + web/classic 两套前端同步支持

### Fixed
- 返利设置保存失败：defaultBillingSettings缺少rebate默认值
- 返利设置GlobalConfig注册：option key统一为rebate_setting.*格式
- PGSQL环境下返利功能失效：key不匹配导致配置无法生效

### Changed
- Dockerfile：添加GOPROXY + 下载重试机制
- docker-compose-prod.yml：镜像名统一为new-api-custom:latest
- docker-compose-mysql.yml：同步command/volumes/env配置
- rebate_setting通过config.GlobalConfig注册（与checkin_setting一致）
