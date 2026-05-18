import React, { useEffect, useState } from 'react';
import {
  Avatar,
  Typography,
  Card,
  Button,
  Space,
  InputNumber,
  Spin,
} from '@douyinfe/semi-ui';
import { Gift, Zap, TrendingUp, Snowflake, AlertTriangle, CheckCircle } from 'lucide-react';
import { API, showError, showSuccess, renderQuota } from '../../helpers';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

const RebateCard = ({ t, renderQuota }) => {
  const { t: tt } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [summary, setSummary] = useState(null);
  const [visible, setVisible] = useState(false);
  const [transferAmount, setTransferAmount] = useState('');
  const [transferring, setTransferring] = useState(false);

  useEffect(() => {
    const checkAndLoad = async () => {
      try {
        const visRes = await API.get('/api/user/rebate/visibility');
        if (visRes.data?.success && visRes.data?.data?.visible) {
          setVisible(true);
          const sumRes = await API.get('/api/user/rebate/summary');
          if (sumRes.data?.success) {
            setSummary(sumRes.data.data);
          }
        }
      } catch (e) {
        // ignore
      }
      setLoading(false);
    };
    checkAndLoad();
  }, []);

  const handleTransfer = async () => {
    const quota = parseInt(transferAmount);
    if (!quota || quota <= 0) return showError(t('请输入有效的划转金额'));
    if (summary && quota > summary.transferable_quota) {
      return showError(t('划转金额超过可划转额度'));
    }
    setTransferring(true);
    try {
      const res = await API.post('/api/user/rebate/transfer', { quota });
      if (res.data?.success) {
        showSuccess(t('划转成功'));
        setTransferAmount('');
        const sumRes = await API.get('/api/user/rebate/summary');
        if (sumRes.data?.success) {
          setSummary(sumRes.data.data);
        }
      } else {
        showError(res.data?.message || t('划转失败'));
      }
    } catch (e) {
      showError(e.message);
    }
    setTransferring(false);
  };

  if (loading) return <Spin />;
  if (!visible) return null;

  return (
    <Card className='!rounded-2xl shadow-sm border-0'>
      <div className='flex items-center mb-4'>
        <Avatar size='small' color='orange' className='mr-3 shadow-md'>
          <Gift size={16} />
        </Avatar>
        <div>
          <Typography.Text className='text-lg font-medium'>
            {t('返利奖励')}
          </Typography.Text>
          <div className='text-xs'>{t('被邀请用户充值时获得返利')}</div>
        </div>
      </div>

      <Space vertical style={{ width: '100%' }}>
        <div className='grid grid-cols-2 sm:grid-cols-4 gap-4 mb-4'>
          <div className='text-center p-3 rounded-xl bg-orange-50'>
            <TrendingUp size={16} className='mx-auto mb-1 text-orange-500' />
            <div className='text-lg font-bold'>
              {renderQuota(summary?.pending_quota || 0)}
            </div>
            <div className='text-xs text-gray-500'>{t('待结算')}</div>
          </div>
          <div className='text-center p-3 rounded-xl bg-green-50'>
            <CheckCircle size={16} className='mx-auto mb-1 text-green-500' />
            <div className='text-lg font-bold'>
              {renderQuota(summary?.settled_quota || 0)}
            </div>
            <div className='text-xs text-gray-500'>{t('可划转')}</div>
          </div>
          <div className='text-center p-3 rounded-xl bg-blue-50'>
            <Zap size={16} className='mx-auto mb-1 text-blue-500' />
            <div className='text-lg font-bold'>
              {renderQuota(summary?.total_transferred_quota || 0)}
            </div>
            <div className='text-xs text-gray-500'>{t('已划转')}</div>
          </div>
          <div className='text-center p-3 rounded-xl bg-red-50'>
            <AlertTriangle size={16} className='mx-auto mb-1 text-red-500' />
            <div className='text-lg font-bold'>
              {renderQuota(summary?.deficit_quota || 0)}
            </div>
            <div className='text-xs text-gray-500'>{t('欠扣')}</div>
          </div>
        </div>

        {summary && summary.frozen_quota > 0 && (
          <div className='flex items-center gap-2 p-2 rounded-lg bg-gray-50 mb-4'>
            <Snowflake size={14} className='text-gray-500' />
            <Text type='tertiary' className='text-sm'>
              {t('冻结')}: {renderQuota(summary.frozen_quota)}
            </Text>
          </div>
        )}

        <Card className='!rounded-xl w-full' title={<Text type='tertiary'>{t('划转返利')}</Text>}>
          <div className='space-y-3'>
            <Text type='tertiary' className='text-sm'>
              {t('将返利奖励划转到您的主余额。')}
            </Text>
            <div className='flex gap-2 items-end'>
              <InputNumber
                value={transferAmount}
                onChange={setTransferAmount}
                placeholder={t('输入划转金额')}
                min={1}
                style={{ width: 200 }}
              />
              <Button
                type='primary'
                theme='solid'
                onClick={handleTransfer}
                loading={transferring}
                disabled={!transferAmount || parseInt(transferAmount) <= 0}
              >
                {t('划转到余额')}
              </Button>
            </div>
          </div>
        </Card>
      </Space>
    </Card>
  );
};

export default RebateCard;
