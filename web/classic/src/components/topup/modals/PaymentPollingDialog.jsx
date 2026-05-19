import React, { useState, useEffect, useCallback } from 'react';
import { Modal, Button, Spin } from '@douyinfe/semi-ui';
import { QRCodeSVG } from 'qrcode.react';
import { useTranslation } from 'react-i18next';
import { API } from '../../../helpers';

const useOrderPolling = (tradeNo, intervalMs = 5000, maxPolls = 60) => {
  const [status, setStatus] = useState(null);
  const [polling, setPolling] = useState(false);

  const stopPolling = useCallback(() => {
    setPolling(false);
  }, []);

  useEffect(() => {
    if (!tradeNo) {
      setPolling(false);
      return;
    }

    setStatus(null);
    setPolling(true);
    let pollCount = 0;
    let stopped = false;

    const timer = setInterval(async () => {
      if (stopped) return;
      pollCount++;
      try {
        const res = await API.get(
          `/api/user/topup/status?trade_no=${encodeURIComponent(tradeNo)}`,
        );
        const s = res.data?.data?.status;
        if (s) {
          setStatus(s);
          if (s === 'success' || s === 'failed' || s === 'expired') {
            stopped = true;
            setPolling(false);
            clearInterval(timer);
            return;
          }
        }
      } catch {
        // ignore
      }
      if (pollCount >= maxPolls) {
        stopped = true;
        setPolling(false);
        setStatus('timeout');
        clearInterval(timer);
      }
    }, intervalMs);

    return () => {
      stopped = true;
      clearInterval(timer);
      setPolling(false);
    };
  }, [tradeNo, intervalMs, maxPolls]);

  return { status, polling, stopPolling };
};

const PaymentPollingDialog = ({
  visible,
  onClose,
  tradeNo,
  paymentType,
  qrCodeUrl,
  onSuccess,
}) => {
  const { t } = useTranslation();
  const { status, polling } = useOrderPolling(tradeNo, 2000);

  useEffect(() => {
    if (status === 'success') {
      if (onSuccess) onSuccess();
      onClose();
    }
  }, [status, onSuccess, onClose]);

  const isWechatNative = paymentType === 'wechat_native';

  const getDescription = () => {
    if (status === 'expired') return t('Order has expired');
    if (status === 'failed') return t('Payment failed');
    if (status === 'timeout')
      return t('Payment result pending, please refresh later');
    if (polling) return t('Waiting for payment result...');
    return '';
  };

  return (
    <Modal
      title={t('Payment Confirmation')}
      visible={visible}
      onCancel={onClose}
      footer={
        <Button onClick={onClose}>{t('关闭')}</Button>
      }
      centered
    >
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          gap: 16,
          padding: '16px 0',
        }}
      >
        <p style={{ color: 'var(--semi-text-2)', fontSize: 14 }}>
          {getDescription()}
        </p>
        {isWechatNative && qrCodeUrl && (
          <div
            style={{
              background: '#fff',
              padding: 16,
              borderRadius: 8,
              border: '1px solid #e0e0e0',
            }}
          >
            <QRCodeSVG value={qrCodeUrl} size={256} level="M" />
          </div>
        )}
        {isWechatNative && (
          <p style={{ color: 'var(--semi-text-2)', fontSize: 14 }}>
            {t('Please scan QR code with WeChat to pay')}
          </p>
        )}
        {!isWechatNative && (
          <p style={{ color: 'var(--semi-text-2)', fontSize: 14 }}>
            {t('Payment page has been opened, please complete payment')}
          </p>
        )}
        {polling && (
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              fontSize: 14,
              color: 'var(--semi-text-2)',
            }}
          >
            <Spin size="small" />
            {t('Waiting for payment result...')}
          </div>
        )}
      </div>
    </Modal>
  );
};

export default PaymentPollingDialog;
