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

import React, { useEffect, useRef, useState } from 'react';
import { API, showError, showSuccess } from '../helpers';
import {
  Button,
  Card,
  Descriptions,
  Form,
  Input,
  InputGroup,
  Modal,
  Space,
  Spin,
  Tag,
} from '@douyinfe/semi-ui';
import { IconPhone, IconKey } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';

const PhoneBindForm = () => {
  const { t } = useTranslation();

  const [loading, setLoading] = useState(true);
  const [status, setStatus] = useState(null);
  const [phone, setPhone] = useState('');
  const [code, setCode] = useState('');
  const [countdown, setCountdown] = useState(0);
  const [sendLoading, setSendLoading] = useState(false);
  const [actionLoading, setActionLoading] = useState(false);
  const [showUnbindModal, setShowUnbindModal] = useState(false);
  const timerRef = useRef(null);

  const fetchStatus = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/phone-auth/status');
      const { success, message, data } = res.data;
      if (success) {
        setStatus(data);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(t('获取手机号状态失败'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchStatus();
    return () => {
      if (timerRef.current) {
        clearInterval(timerRef.current);
      }
    };
  }, []);

  const handlePhoneChange = (value) => {
    const digits = value.replace(/\D/g, '');
    if (digits.length <= 11) {
      setPhone(digits);
    }
  };

  const handleCodeChange = (value) => {
    const digits = value.replace(/\D/g, '');
    if (digits.length <= 6) {
      setCode(digits);
    }
  };

  const handleSendCode = async (targetPhone) => {
    const phoneNumber = targetPhone || phone;
    if (!phoneNumber || phoneNumber.length !== 11) {
      showError(t('请输入正确的11位手机号'));
      return;
    }

    setSendLoading(true);
    try {
      const res = await API.post('/api/phone-auth/sms/send', { phone: phoneNumber });
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('验证码已发送'));
        setCountdown(60);
        timerRef.current = setInterval(() => {
          setCountdown((prev) => {
            if (prev <= 1) {
              clearInterval(timerRef.current);
              return 0;
            }
            return prev - 1;
          });
        }, 1000);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(t('发送验证码失败，请重试'));
    } finally {
      setSendLoading(false);
    }
  };

  const handleBind = async () => {
    if (!phone || phone.length !== 11) {
      showError(t('请输入正确的11位手机号'));
      return;
    }
    if (!code || code.length !== 6) {
      showError(t('请输入6位验证码'));
      return;
    }

    setActionLoading(true);
    try {
      const res = await API.post('/api/phone-auth/bind', { phone, code });
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('绑定成功'));
        setPhone('');
        setCode('');
        fetchStatus();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(t('绑定失败，请重试'));
    } finally {
      setActionLoading(false);
    }
  };

  const handleRebind = async () => {
    if (!phone || phone.length !== 11) {
      showError(t('请输入正确的11位手机号'));
      return;
    }
    if (!code || code.length !== 6) {
      showError(t('请输入6位验证码'));
      return;
    }

    setActionLoading(true);
    try {
      const res = await API.post('/api/phone-auth/rebind', { phone, code });
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('换绑成功'));
        setPhone('');
        setCode('');
        fetchStatus();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(t('换绑失败，请重试'));
    } finally {
      setActionLoading(false);
    }
  };

  const handleUnbind = async () => {
    setActionLoading(true);
    try {
      const res = await API.post('/api/phone-auth/unbind');
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('解绑成功'));
        setShowUnbindModal(false);
        fetchStatus();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(t('解绑失败，请重试'));
    } finally {
      setActionLoading(false);
    }
  };

  const maskPhone = (p) => {
    if (!p || p.length < 7) return p;
    return p.slice(0, 3) + '****' + p.slice(7);
  };

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: 40 }}>
        <Spin size='large' />
      </div>
    );
  }

  const isBound = status?.phone_bound;
  const isVerified = status?.real_name_verified;

  return (
    <Card title={t('手机号认证')} style={{ maxWidth: 600 }}>
      {isBound && (
        <Descriptions
          row
          data={[
            {
              key: t('当前手机号'),
              value: maskPhone(status.phone),
            },
            {
              key: t('实名认证状态'),
              value: isVerified ? (
                <Tag color='green'>{t('已认证')}</Tag>
              ) : (
                <Tag color='orange'>{t('未认证')}</Tag>
              ),
            },
          ]}
          style={{ marginBottom: 24 }}
        />
      )}

      {isBound ? (
        <>
          <Form className='space-y-3'>
            <Form.Input
              field='phone'
              label={t('新手机号')}
              placeholder={t('请输入新手机号')}
              value={phone}
              onChange={handlePhoneChange}
              prefix={<IconPhone />}
              maxLength={11}
            />

            <div>
              <label className='semi-form-field-label-text'>
                {t('验证码')}
              </label>
              <InputGroup>
                <Input
                  placeholder={t('请输入验证码')}
                  value={code}
                  onChange={handleCodeChange}
                  prefix={<IconKey />}
                  maxLength={6}
                  style={{ flex: 1 }}
                />
                <Button
                  theme='solid'
                  onClick={() => handleSendCode(phone)}
                  loading={sendLoading}
                  disabled={countdown > 0 || phone.length !== 11}
                  style={{ minWidth: 120 }}
                >
                  {countdown > 0
                    ? t('{{count}}秒后重试', { count: countdown })
                    : t('发送验证码')}
                </Button>
              </InputGroup>
            </div>

            <Space style={{ marginTop: 12 }}>
              <Button
                theme='solid'
                type='primary'
                onClick={handleRebind}
                loading={actionLoading}
              >
                {t('换绑手机号')}
              </Button>
              <Button
                type='danger'
                onClick={() => setShowUnbindModal(true)}
              >
                {t('解绑手机号')}
              </Button>
            </Space>
          </Form>
        </>
      ) : (
        <>
          <Form className='space-y-3'>
            <Form.Input
              field='phone'
              label={t('手机号')}
              placeholder={t('请输入手机号')}
              value={phone}
              onChange={handlePhoneChange}
              prefix={<IconPhone />}
              maxLength={11}
            />

            <div>
              <label className='semi-form-field-label-text'>
                {t('验证码')}
              </label>
              <InputGroup>
                <Input
                  placeholder={t('请输入验证码')}
                  value={code}
                  onChange={handleCodeChange}
                  prefix={<IconKey />}
                  maxLength={6}
                  style={{ flex: 1 }}
                />
                <Button
                  theme='solid'
                  onClick={() => handleSendCode(phone)}
                  loading={sendLoading}
                  disabled={countdown > 0 || phone.length !== 11}
                  style={{ minWidth: 120 }}
                >
                  {countdown > 0
                    ? t('{{count}}秒后重试', { count: countdown })
                    : t('发送验证码')}
                </Button>
              </InputGroup>
            </div>

            <Button
              theme='solid'
              className='w-full'
              type='primary'
              onClick={handleBind}
              loading={actionLoading}
              style={{ marginTop: 12 }}
            >
              {t('绑定手机号')}
            </Button>
          </Form>
        </>
      )}

      <Modal
        title={t('确认解绑')}
        visible={showUnbindModal}
        onOk={handleUnbind}
        onCancel={() => setShowUnbindModal(false)}
        okText={t('确认解绑')}
        cancelText={t('取消')}
        okButtonProps={{ loading: actionLoading, type: 'danger' }}
        centered
      >
        <p>{t('解绑后将无法使用手机号登录，确认解绑？')}</p>
      </Modal>
    </Card>
  );
};

export default PhoneBindForm;
