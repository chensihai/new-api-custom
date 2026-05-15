import React, { useEffect, useState, useRef } from 'react';
import { Banner, Button, Form, Row, Col, Spin } from '@douyinfe/semi-ui';
import {
  API,
  removeTrailingSlash,
  showError,
  showSuccess,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';
import { BookOpen } from 'lucide-react';

const toBoolean = (value) => value === true || value === 'true';

export default function SettingsPaymentGatewayWechat(props) {
  const { t } = useTranslation();
  const sectionTitle = props.hideSectionTitle ? undefined : t('微信支付设置');
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    WechatPayEnabled: false,
    WechatPayMchID: '',
    WechatPayAPIv3Key: '',
    WechatPaySerialNo: '',
    WechatPayPrivateKey: '',
    WechatPayMinTopUp: 1,
  });
  const [originInputs, setOriginInputs] = useState({});
  const formApiRef = useRef(null);

  useEffect(() => {
    if (props.options && formApiRef.current) {
      const currentInputs = {
        WechatPayEnabled: toBoolean(props.options.WechatPayEnabled),
        WechatPayMchID: props.options.WechatPayMchID || '',
        WechatPayAPIv3Key: props.options.WechatPayAPIv3Key || '',
        WechatPaySerialNo: props.options.WechatPaySerialNo || '',
        WechatPayPrivateKey: props.options.WechatPayPrivateKey || '',
        WechatPayMinTopUp: parseInt(props.options.WechatPayMinTopUp) || 1,
      };
      setInputs(currentInputs);
      setOriginInputs({ ...currentInputs });
      formApiRef.current.setValues(currentInputs);
    }
  }, [props.options]);

  const handleFormChange = (values) => {
    setInputs(values);
  };

  const submitWechatPaySetting = async () => {
    if (props.options.ServerAddress === '') {
      showError(t('请先填写服务器地址'));
      return;
    }

    setLoading(true);
    try {
      const options = [];

      if (originInputs.WechatPayEnabled !== inputs.WechatPayEnabled) {
        options.push({
          key: 'WechatPayEnabled',
          value: inputs.WechatPayEnabled ? 'true' : 'false',
        });
      }
      if (originInputs.WechatPayMchID !== inputs.WechatPayMchID) {
        options.push({ key: 'WechatPayMchID', value: inputs.WechatPayMchID || '' });
      }
      if (inputs.WechatPayAPIv3Key && inputs.WechatPayAPIv3Key !== '') {
        options.push({ key: 'WechatPayAPIv3Key', value: inputs.WechatPayAPIv3Key });
      }
      if (originInputs.WechatPaySerialNo !== inputs.WechatPaySerialNo) {
        options.push({ key: 'WechatPaySerialNo', value: inputs.WechatPaySerialNo || '' });
      }
      if (inputs.WechatPayPrivateKey && inputs.WechatPayPrivateKey !== '') {
        options.push({ key: 'WechatPayPrivateKey', value: inputs.WechatPayPrivateKey });
      }
      if (originInputs.WechatPayMinTopUp !== inputs.WechatPayMinTopUp) {
        options.push({
          key: 'WechatPayMinTopUp',
          value: String(inputs.WechatPayMinTopUp || 1),
        });
      }

      if (options.length === 0) {
        showSuccess(t('没有需要更新的配置'));
        setLoading(false);
        return;
      }

      const requestQueue = options.map((opt) =>
        API.put('/api/option/', {
          key: opt.key,
          value: opt.value,
        }),
      );

      const results = await Promise.all(requestQueue);

      const errorResults = results.filter((res) => !res.data.success);
      if (errorResults.length > 0) {
        errorResults.forEach((res) => {
          showError(res.data.message);
        });
      } else {
        showSuccess(t('更新成功'));
        props.refresh?.();
      }
    } catch (error) {
      showError(t('更新失败'));
    }
    setLoading(false);
  };

  return (
    <Spin spinning={loading}>
      <Form
        initValues={inputs}
        onValueChange={handleFormChange}
        getFormApi={(api) => (formApiRef.current = api)}
      >
        <Form.Section text={sectionTitle}>
          <Banner
            type='info'
            icon={<BookOpen size={16} />}
            description={
              <>
                {t('微信支付商户和密钥设置请前往')}
                <a
                  href='https://pay.weixin.qq.com'
                  target='_blank'
                  rel='noreferrer'
                >
                  {t('微信支付商户平台')}
                </a>
                {t('进行配置，支持 Native 扫码支付和 H5 支付。')}
                <br />
                {t('回调地址')}：
                {props.options.ServerAddress
                  ? removeTrailingSlash(props.options.ServerAddress)
                  : t('网站地址')}
                /api/wechat/notify
              </>
            }
            style={{ marginBottom: 16 }}
          />

          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Switch
                field='WechatPayEnabled'
                label={t('启用微信支付')}
                size='default'
                checkedText='｜'
                uncheckedText='〇'
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='WechatPayMchID'
                label={t('商户号')}
                placeholder={t('例如：16xxxxxx')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.InputNumber
                field='WechatPayMinTopUp'
                label={t('最低充值数量')}
                placeholder={t('例如：1')}
                min={1}
              />
            </Col>
          </Row>

          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{ marginTop: 16 }}
          >
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='WechatPaySerialNo'
                label={t('证书序列号')}
                placeholder={t('例如：5A2xxxxxx')}
                extraText={t('商户API证书的序列号')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='WechatPayAPIv3Key'
                label={t('APIv3 密钥')}
                placeholder={t('填写后覆盖当前密钥，留空表示保持当前不变')}
                extraText={t('用于回调验签和解密，保存后不会回显')}
                type='password'
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.TextArea
                field='WechatPayPrivateKey'
                label={t('商户私钥')}
                placeholder={t('填写后覆盖当前私钥，留空表示保持当前不变')}
                extraText={t('PEM 格式私钥，保存后不会回显')}
                type='password'
                autosize={{ minRows: 3, maxRows: 6 }}
              />
            </Col>
          </Row>

          <Button onClick={submitWechatPaySetting} style={{ marginTop: 16 }}>
            {t('更新微信支付设置')}
          </Button>
        </Form.Section>
      </Form>
    </Spin>
  );
}
