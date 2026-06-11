import React, { useContext, useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { UserContext } from '../../context/User';
import { API, showError, showSuccess, setUserData, updateAPI } from '../../helpers';
import { Button, Checkbox, Form, Input, InputGroup, Typography } from '@douyinfe/semi-ui';
import { IconPhone, IconKey, IconUser, IconLock } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';

const PhoneRegisterForm = ({ agreedToTerms, setAgreedToTerms, hasUserAgreement, hasPrivacyPolicy }) => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [userState, userDispatch] = useContext(UserContext);

  const [phone, setPhone] = useState('');
  const [code, setCode] = useState('');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [countdown, setCountdown] = useState(0);
  const [sendLoading, setSendLoading] = useState(false);
  const [registerLoading, setRegisterLoading] = useState(false);
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

  const handleRegister = async () => {
    if (!phone || phone.length !== 11) {
      showError(t('请输入正确的11位手机号'));
      return;
    }
    if (!code || code.length !== 6) {
      showError(t('请输入6位验证码'));
      return;
    }
    if (!username.trim()) {
      showError(t('用户名不能为空'));
      return;
    }
    if ((hasUserAgreement || hasPrivacyPolicy) && !agreedToTerms) {
      showError(t('请先阅读并同意用户协议和隐私政策'));
      return;
    }

    setRegisterLoading(true);
    try {
      const payload = { phone, code, username: username.trim() };
      if (password) {
        payload.password = password;
      }
      const res = await API.post('/api/phone-auth/register', payload);
      const { success, message, data } = res.data;
      if (success) {
        userDispatch({ type: 'login', payload: data });
        setUserData(data);
        updateAPI();
        showSuccess(t('注册成功'));
        navigate('/console');
      } else {
        showError(message);
      }
    } catch (error) {
      showError(t('注册失败，请重试'));
    } finally {
      setRegisterLoading(false);
    }
  };

  return (
    <Form className='space-y-3'>
      <Form.Input
        field='phone'
        label={t('手机号')}
        placeholder={t('请输入11位手机号')}
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

      <Form.Input
        field='username'
        label={t('请输入用户名')}
        placeholder={t('用户名')}
        value={username}
        onChange={setUsername}
        prefix={<IconUser />}
        showClear
      />

      <Form.Input
        field='password'
        label={t('密码')}
        placeholder={t('设置密码（可选）')}
        value={password}
        onChange={setPassword}
        mode='password'
        prefix={<IconLock />}
      />

      {(hasUserAgreement || hasPrivacyPolicy) && (
        <div className='pt-2'>
          <Checkbox
            checked={agreedToTerms}
            onChange={(e) => setAgreedToTerms && setAgreedToTerms(e.target.checked)}
          >
            <Typography.Text size='small' className='text-gray-600'>
              {t('我已阅读并同意')}
              {hasUserAgreement && (
                <>
                  <a href='/user-agreement' target='_blank' rel='noopener noreferrer' className='text-blue-600 hover:text-blue-800 mx-1'>
                    {t('用户协议')}
                  </a>
                </>
              )}
              {hasUserAgreement && hasPrivacyPolicy && t('和')}
              {hasPrivacyPolicy && (
                <>
                  <a href='/privacy-policy' target='_blank' rel='noopener noreferrer' className='text-blue-600 hover:text-blue-800 mx-1'>
                    {t('隐私政策')}
                  </a>
                </>
              )}
            </Typography.Text>
          </Checkbox>
        </div>
      )}

      <Button
        theme='solid'
        className='w-full !rounded-full'
        type='primary'
        htmlType='submit'
        onClick={handleRegister}
        loading={registerLoading}
        disabled={
          (hasUserAgreement || hasPrivacyPolicy) && !agreedToTerms
        }
        style={{ marginTop: 12 }}
      >
        {t('注册')}
      </Button>
    </Form>
  );
};

export default PhoneRegisterForm;
