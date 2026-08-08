import { ReactElement, useEffect, useState } from 'react';
import { LockOutlined, LoginOutlined, UserOutlined } from '@ant-design/icons';
import { Alert, Button, Divider, Form, Input, Spin } from 'antd';
import { useTranslation } from 'react-i18next';
import { useNavigate, useSearchParams } from 'react-router-dom';

import * as api from '@/api/auth.ts';
import { encrypt } from '@/lib/encrypt.ts';
import { Head } from '@/components/head.tsx';

import { Tips } from './tips.tsx';

export const Login = (): ReactElement => {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { t } = useTranslation();

  const [isLoading, setIsloading] = useState(false);
  const [msg, setMsg] = useState('');
  const [config, setConfig] = useState<api.AuthConfig | null>(null);

  const oidcError = searchParams.get('oidc_error');
  const oidcErrorMessage = getOIDCErrorMessage(oidcError, t);

  useEffect(() => {
    let active = true;

    api
      .getSession()
      .then((rsp) => {
        if (active && rsp.code === 0 && rsp.data.authenticated) {
          navigate('/', { replace: true });
        }
      })
      .catch(() => undefined);

    api
      .getConfig()
      .then((rsp) => {
        if (active && rsp.code === 0) setConfig(rsp.data);
        else if (active) setConfig(localFallbackConfig);
      })
      .catch(() => {
        if (active) setConfig(localFallbackConfig);
      });

    return () => {
      active = false;
    };
  }, [navigate]);

  useEffect(() => {
    if (msg) {
      const timer = window.setTimeout(() => setMsg(''), 3000);
      return () => window.clearTimeout(timer);
    }
  }, [msg]);

  function login(values: any) {
    if (isLoading) return;
    setIsloading(true);

    const username = values.username;
    const password = encrypt(values.password);

    api
      .login(username, password)
      .then((rsp: any) => {
        if (rsp.code !== 0) {
          let errorMsg = t('auth.error');
          if (rsp.code === -2) errorMsg = t('auth.invalidUser');
          else if (rsp.code === -5) errorMsg = t('auth.locked');
          else if (rsp.code === -4) errorMsg = t('auth.globalLocked');

          setMsg(errorMsg);
          return;
        }

        setMsg('');
        navigate('/', { replace: true });
        window.location.reload();
      })
      .catch(() => {
        setMsg(t('auth.error'));
      })
      .finally(() => {
        setIsloading(false);
      });
  }

  return (
    <>
      <Head title={t('head.login')} />

      <div className="flex h-screen w-screen flex-col items-center justify-center">
        <div style={{ minWidth: 300, maxWidth: 500 }}>
          <div className="flex flex-col items-center justify-center pb-4">
            <img
              id="logo"
              src="/sipeed.ico"
              alt="Sipeed"
              onClick={(evt) => {
                evt.preventDefault();
                (evt.target as HTMLImageElement).classList.add('animate-spin');
                setTimeout(() => {
                  (evt.target as HTMLImageElement).classList.remove('animate-spin');
                }, 1000);
              }}
            />
          </div>
          {oidcErrorMessage && (
            <Alert className="mb-4" type="error" showIcon message={oidcErrorMessage} />
          )}

          {!config ? (
            <div className="flex justify-center py-8" role="status">
              <Spin />
              <span className="sr-only">{t('auth.loadingAuth')}</span>
            </div>
          ) : (
            <>
              {config.oidcEnabled && (
                <>
                  {config.oidcReady ? (
                    <Button
                      className="w-full"
                      type="primary"
                      size="large"
                      icon={<LoginOutlined />}
                      href="/api/auth/oidc/login"
                    >
                      {t('auth.oidcLogin', {
                        provider: config.providerName || t('auth.oidcDefaultProvider')
                      })}
                    </Button>
                  ) : (
                    <Button
                      className="w-full"
                      type="primary"
                      size="large"
                      icon={<LoginOutlined />}
                      disabled
                    >
                      {t('auth.oidcLogin', {
                        provider: config.providerName || t('auth.oidcDefaultProvider')
                      })}
                    </Button>
                  )}
                  {!config.oidcReady && (
                    <Alert
                      className="mt-4"
                      type="error"
                      showIcon
                      message={
                        config.oidcError === 'invalid_configuration'
                          ? t('auth.oidcInvalidConfig')
                          : t('auth.oidcUnavailable')
                      }
                    />
                  )}
                </>
              )}

              {config.oidcEnabled && config.allowLocalLogin && (
                <Divider plain>{t('auth.oidcOrLocal')}</Divider>
              )}

              {config.allowLocalLogin && (
                <Form initialValues={{ remember: true }} onFinish={login}>
                  <Form.Item
                    name="username"
                    rules={[{ required: true, message: t('auth.noEmptyUsername'), min: 1 }]}
                  >
                    <Input
                      prefix={<UserOutlined />}
                      placeholder={t('auth.placeholderUsername')}
                      aria-label={t('auth.placeholderUsername')}
                    />
                  </Form.Item>

                  <Form.Item
                    name="password"
                    rules={[{ required: true, message: t('auth.noEmptyPassword'), min: 1 }]}
                  >
                    <Input
                      prefix={<LockOutlined />}
                      type="password"
                      placeholder={t('auth.placeholderPassword')}
                      aria-label={t('auth.placeholderPassword')}
                    />
                  </Form.Item>

                  {msg && (
                    <div className="pb-1 text-red-500" role="alert">
                      {msg}
                    </div>
                  )}

                  <Form.Item>
                    <Button type="primary" htmlType="submit" className="w-full" loading={isLoading}>
                      {t('auth.loginButtonText')}
                    </Button>
                  </Form.Item>

                  <div className="flex justify-end pb-4 text-sm">
                    <Tips />
                  </div>
                </Form>
              )}

              {!config.oidcEnabled && !config.allowLocalLogin && (
                <Alert type="error" showIcon message={t('auth.noLoginMethods')} />
              )}
            </>
          )}
        </div>
      </div>
    </>
  );
};

const localFallbackConfig: api.AuthConfig = {
  oidcEnabled: false,
  oidcReady: false,
  providerName: '',
  allowLocalLogin: true
};

function getOIDCErrorMessage(error: string | null, t: (key: string) => string): string {
  switch (error) {
    case 'oidc_access_denied':
      return t('auth.oidcAccessDenied');
    case 'oidc_disabled':
      return t('auth.oidcDisabled');
    case 'oidc_invalid_config':
      return t('auth.oidcInvalidConfig');
    case 'oidc_provider_unavailable':
      return t('auth.oidcUnavailable');
    case 'oidc_rate_limited':
      return t('auth.locked');
    case null:
      return '';
    default:
      return t('auth.oidcLoginFailed');
  }
}
