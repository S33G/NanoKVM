import type { ReactNode } from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';

import { Logout } from '@/pages/desktop/menu/settings/account/logout.tsx';

const mocks = vi.hoisted(() => ({ logout: vi.fn(), removeToken: vi.fn() }));
vi.mock('@/api/auth.ts', () => ({ logout: mocks.logout }));
vi.mock('@/lib/cookie.ts', () => ({ removeToken: mocks.removeToken }));
vi.mock('react-i18next', () => ({ useTranslation: () => ({ t: (key: string) => key }) }));
vi.mock('@ant-design/icons', () => ({ LogoutOutlined: () => null }));
vi.mock('antd', () => ({
  Button: ({ children }: { children: ReactNode }) => <button>{children}</button>,
  Popconfirm: ({ children, onConfirm }: { children: ReactNode; onConfirm: () => void }) => (
    <div onClick={onConfirm}>{children}</div>
  )
}));

describe('Logout', () => {
  it('calls server logout, clears a legacy token, and replaces the route', async () => {
    mocks.logout.mockResolvedValue({ code: 0, msg: '', data: null });

    render(
      <MemoryRouter initialEntries={['/']}>
        <Routes>
          <Route path="/" element={<Logout />} />
          <Route path="/auth/login" element={<>login page</>} />
        </Routes>
      </MemoryRouter>
    );

    fireEvent.click(screen.getByRole('button'));

    expect(await screen.findByText('login page')).toBeInTheDocument();
    expect(mocks.logout).toHaveBeenCalledOnce();
    expect(mocks.removeToken).toHaveBeenCalledOnce();
  });
});
