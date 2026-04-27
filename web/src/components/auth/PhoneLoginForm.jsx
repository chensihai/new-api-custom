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

import React, { useContext, useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { UserContext } from '../../context/User';
import { API, showError, showSuccess, setUserData, updateAPI } from '../../helpers';
import { Button, Form, Input, InputGroup } from '@douyinfe/semi-ui';
import { IconPhone, IconKey } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';

const PhoneLoginForm = ({ agreedToTerms, hasUserAgreement, hasPrivacyPolicy }) => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [userState, userDispatch] = useContext(UserContext);

  const [phone, setPhone] = useState('');
  const [code, setCode] = useState('');
  const [countdown, setCountdown] = useState(0);
  const [sendLoading, setSendLoading] = useState(false);
  const [loginLoading, setLoginLoading] = useState(false);
  const timerRef = useRef(null);

  useEffect(() => {
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

  const handleSendCode = async () => {
    if (!phone || phone.length !== 11) {
      showError(t('请输入正确的11位手机号'));
      return;
    }
    if ((hasUserAgreement || hasPrivacyPolicy) && !agreedToTerms) {
      showError(t('请先阅读并同意用户协议和隐私政策'));
      return;
    }

    setSendLoading(true);
    try {
      const res = await API.post('/api/phone-auth/sms/send', { phone });
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

  const handleLogin = async () => {
    if (!phone || phone.length !== 11) {
      showError(t('请输入正确的11位手机号'));
      return;
    }
    if (!code || code.length !== 6) {
      showError(t('请输入6位验证码'));
      return;
    }
    if ((hasUserAgreement || hasPrivacyPolicy) && !agreedToTerms) {
      showError(t('请先阅读并同意用户协议和隐私政策'));
      return;
    }

    setLoginLoading(true);
    try {
      const res = await API.post('/api/phone-auth/sms/login', { phone, code });
      const { success, message, data } = res.data;
      if (success) {
        userDispatch({ type: 'login', payload: data });
        setUserData(data);
        updateAPI();
        showSuccess(t('登录成功'));
        navigate('/console');
      } else {
        showError(message);
      }
    } catch (error) {
      showError(t('登录失败，请重试'));
    } finally {
      setLoginLoading(false);
    }
  };

  return (
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
            onClick={handleSendCode}
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
        className='w-full !rounded-full'
        type='primary'
        htmlType='submit'
        onClick={handleLogin}
        loading={loginLoading}
        disabled={
          (hasUserAgreement || hasPrivacyPolicy) && !agreedToTerms
        }
        style={{ marginTop: 12 }}
      >
        {t('登录')}
      </Button>
    </Form>
  );
};

export default PhoneLoginForm;
