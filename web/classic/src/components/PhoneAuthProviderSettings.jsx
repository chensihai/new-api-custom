/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useEffect, useState } from 'react';
import { API, showError, showSuccess } from '../helpers';
import {
  Button,
  Card,
  Input,
  Switch,
  Spin,
  Tabs,
  Typography,
} from '@douyinfe/semi-ui';

const { Text } = Typography;

const PROVIDERS = [
  { key: 'aliyun', label: '阿里云' },
  { key: 'tencent', label: '腾讯云' },
  { key: 'huawei', label: '华为云' },
];

const PROVIDER_FIELDS = {
  aliyun: [
    { key: 'aliyun_access_key_id', label: 'AccessKey ID', secret: false },
    { key: 'aliyun_access_key_secret', label: 'AccessKey Secret', secret: true },
    { key: 'aliyun_sign_name', label: '短信签名', secret: false },
    { key: 'aliyun_template_code', label: '模板Code', secret: false },
  ],
  tencent: [
    { key: 'tencent_secret_id', label: 'SecretId', secret: false },
    { key: 'tencent_secret_key', label: 'SecretKey', secret: true },
    { key: 'tencent_sms_sdk_app_id', label: 'SdkAppId', secret: false },
    { key: 'tencent_sign_name', label: '短信签名', secret: false },
    { key: 'tencent_template_id', label: '模板ID', secret: false },
  ],
  huawei: [
    { key: 'huawei_app_id', label: 'APP Key', secret: false },
    { key: 'huawei_app_secret', label: 'APP Secret', secret: true },
    { key: 'huawei_sign_name', label: '短信签名', secret: false },
    { key: 'huawei_template_id', label: '模板ID', secret: false },
    { key: 'huawei_sender', label: '国内短信发送通道号', secret: false },
  ],
};

const ENABLED_KEYS = {
  aliyun: 'aliyun_enabled',
  tencent: 'tencent_enabled',
  huawei: 'huawei_enabled',
};

const PhoneAuthProviderSettings = ({ t: tProp }) => {
  const t = tProp;

  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [activeTab, setActiveTab] = useState('aliyun');
  const [settings, setSettings] = useState({});

  const getEnabledProvider = () => {
    return PROVIDERS.find((p) => settings[ENABLED_KEYS[p.key]])?.key || null;
  };

  const fetchProviders = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/phone-auth/admin/providers');
      const { success, message, data } = res.data;
      if (success && data && data.settings) {
        setSettings(data.settings);
      } else if (!success) {
        showError(message);
      }
    } catch (error) {
      showError(t('获取认证服务商配置失败'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchProviders();
  }, []);

  const handleToggleProvider = (key, checked) => {
    setSettings((prev) => {
      const updated = { ...prev };
      if (checked) {
        for (const p of PROVIDERS) {
          updated[ENABLED_KEYS[p.key]] = p.key === key;
        }
      } else {
        updated[ENABLED_KEYS[key]] = false;
      }
      return updated;
    });
  };

  const handleFieldChange = (fieldKey, value) => {
    setSettings((prev) => ({ ...prev, [fieldKey]: value }));
  };

  const checkProviderComplete = (key) => {
    const fields = PROVIDER_FIELDS[key] || [];
    return fields.every((f) => settings[f.key] && String(settings[f.key]).trim() !== '');
  };

  const handleSave = async () => {
    const enabledKey = getEnabledProvider();
    if (enabledKey && !checkProviderComplete(enabledKey)) {
      const provider = PROVIDERS.find((p) => p.key === enabledKey);
      showError(t('{{provider}}凭证配置不完整，请补全后保存', { provider: provider.label }));
      setActiveTab(enabledKey);
      return;
    }

    setSaving(true);
    try {
      const payload = { ...settings };
      const res = await API.put('/api/phone-auth/admin/providers', payload);
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('保存成功'));
        await fetchProviders();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(t('保存失败，请重试'));
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: 40 }}>
        <Spin size='large' />
      </div>
    );
  }

  const currentEnabled = getEnabledProvider();

  return (
    <Card title={t('手机号认证服务商配置')}>
      <Tabs activeKey={activeTab} onChange={setActiveTab}>
        {PROVIDERS.map((provider) => {
          const isEnabled = settings[ENABLED_KEYS[provider.key]] || false;
          const isComplete = checkProviderComplete(provider.key);
          const fields = PROVIDER_FIELDS[provider.key] || [];
          const isOtherEnabled = currentEnabled && currentEnabled !== provider.key;

          return (
            <Tabs.TabPane
              key={provider.key}
              itemKey={provider.key}
              tab={
                <span style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                  {provider.label}
                  {isEnabled && !isComplete && (
                    <Text type='warning' size='small'>
                      ({t('凭证不完整')})
                    </Text>
                  )}
                  {isEnabled && isComplete && (
                    <Text type='success' size='small'>
                      ✓
                    </Text>
                  )}
                </span>
              }
            >
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  marginBottom: 16,
                  marginTop: 8,
                }}
              >
                <Text strong>{t('启用{{provider}}', { provider: provider.label })}</Text>
                <Switch
                  checked={isEnabled}
                  onChange={(checked) =>
                    handleToggleProvider(provider.key, checked)
                  }
                  disabled={isOtherEnabled}
                />
              </div>

              {isOtherEnabled && (
                <Text type='tertiary' style={{ marginBottom: 12, display: 'block' }}>
                  {t('已选择其他服务商，同一时间仅支持启用一个短信服务商')}
                </Text>
              )}

              {isEnabled && (
                <div>
                  {fields.map((field) => (
                    <div key={field.key} style={{ marginBottom: 12 }}>
                      <label
                        style={{
                          display: 'block',
                          marginBottom: 4,
                          color: 'var(--semi-color-text-0)',
                          fontWeight: 600,
                          fontSize: 14,
                        }}
                      >
                        {field.label}
                      </label>
                      <Input
                        placeholder={field.label}
                        value={settings[field.key] || ''}
                        onChange={(value) =>
                          handleFieldChange(field.key, value)
                        }
                        mode={field.secret ? 'password' : undefined}
                      />
                    </div>
                  ))}
                </div>
              )}
            </Tabs.TabPane>
          );
        })}
      </Tabs>

      <div style={{ marginTop: 24, textAlign: 'right' }}>
        <Button theme='solid' type='primary' onClick={handleSave} loading={saving}>
          {t('保存配置')}
        </Button>
      </div>
    </Card>
  );
};

export default PhoneAuthProviderSettings;
