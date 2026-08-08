import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { Login } from '@/pages/auth/login';

const auth = vi.hoisted(() => ({
  getConfig: vi.fn(),
  getSession: vi.fn(),
  login: vi.fn()
}));

vi.mock('@/api/auth.ts', () => auth);
vi.mock('@/components/head.tsx', () => ({ Head: () => null }));
vi.mock('@/pages/auth/login/tips.tsx', () => ({ Tips: () => null }));
vi.mock('@/lib/encrypt.ts', () => ({ encrypt: (value: string) => value }));
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, values?: { provider?: string }) => {
      const strings: Record<string, string> = {
        'auth.loginButtonText': 'Login',
        'auth.placeholderUsername': 'Username',
        'auth.placeholderPassword': 'Password',
        'auth.oidcLogin': `Continue with ${values?.provider}`,
        'auth.oidcOrLocal': 'or use a local account',
        'auth.oidcUnavailable': 'Single sign-on is currently unavailable.',
        'auth.oidcDisabled': 'Single sign-on is not enabled on this device.',
        'auth.oidcInvalidConfig': 'Single sign-on configuration is invalid.',
        'auth.oidcLoginFailed': 'Single sign-on failed. Please try again.',
        'auth.oidcAccessDenied': 'Single sign-on was cancelled or denied.',
        'auth.locked': 'Too many logins, please try again later',
        'auth.noLoginMethods': 'No sign-in methods are available.'
      };
      return strings[key] || key;
    }
  })
}));

const unauthenticated = {
  code: 0,
  msg: '',
  data: { authenticated: false, username: '', authSource: '', admin: false }
};

describe('Login', () => {
  beforeEach(() => {
    auth.getSession.mockResolvedValue(unauthenticated);
  });

  it('shows a root-relative OIDC link and hides local login when configured', async () => {
    auth.getConfig.mockResolvedValue({
      code: 0,
      msg: '',
      data: {
        oidcEnabled: true,
        oidcReady: true,
        providerName: 'Example ID',
        allowLocalLogin: false
      }
    });

    renderLogin();

    const link = await screen.findByRole('link', { name: /continue with example id/i });
    expect(link).toHaveAttribute('href', '/api/auth/oidc/login');
    expect(screen.queryByLabelText('Username')).not.toBeInTheDocument();
  });

  it('keeps local login available when auth config cannot be loaded', async () => {
    auth.getConfig.mockRejectedValue(new Error('unavailable'));

    renderLogin();

    expect(await screen.findByLabelText('Username')).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: /continue with/i })).not.toBeInTheDocument();
  });

  it('shows both configured methods and disables an unready provider', async () => {
    auth.getConfig.mockResolvedValue({
      code: 0,
      msg: '',
      data: {
        oidcEnabled: true,
        oidcReady: false,
        oidcError: 'invalid_configuration',
        providerName: 'Example ID',
        allowLocalLogin: true
      }
    });

    renderLogin();

    expect(await screen.findByRole('button', { name: /continue with example id/i })).toBeDisabled();
    expect(screen.getByLabelText('Username')).toBeInTheDocument();
    expect(screen.getByText('Single sign-on configuration is invalid.')).toBeInTheDocument();
  });

  it('renders a safe callback error without exposing its code', async () => {
    auth.getConfig.mockResolvedValue({
      code: 0,
      msg: '',
      data: {
        oidcEnabled: false,
        oidcReady: false,
        providerName: '',
        allowLocalLogin: true
      }
    });

    renderLogin('/auth/login?oidc_error=secret_internal_detail');

    expect(await screen.findByText('Single sign-on failed. Please try again.')).toBeInTheDocument();
    expect(screen.queryByText('secret_internal_detail')).not.toBeInTheDocument();
  });

  it('maps group-policy denial to a safe access-denied message', async () => {
    auth.getConfig.mockResolvedValue({
      code: 0,
      msg: '',
      data: {
        oidcEnabled: true,
        oidcReady: true,
        providerName: 'Example ID',
        allowLocalLogin: true
      }
    });

    renderLogin('/auth/login?oidc_error=oidc_access_denied');

    expect(await screen.findByText('Single sign-on was cancelled or denied.')).toBeInTheDocument();
  });

  it('maps provider outages and rate limits to useful safe errors', async () => {
    auth.getConfig.mockResolvedValue({
      code: 0,
      msg: '',
      data: {
        oidcEnabled: true,
        oidcReady: true,
        providerName: 'Example ID',
        allowLocalLogin: true
      }
    });

    const { unmount } = renderLogin('/auth/login?oidc_error=oidc_provider_unavailable');
    expect(await screen.findByText('Single sign-on is currently unavailable.')).toBeInTheDocument();
    unmount();

    renderLogin('/auth/login?oidc_error=oidc_rate_limited');
    expect(await screen.findByText('Too many logins, please try again later')).toBeInTheDocument();
  });
});

function renderLogin(path = '/auth/login') {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Login />
    </MemoryRouter>
  );
}
