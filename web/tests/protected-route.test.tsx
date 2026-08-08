import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';

import { ProtectedRoute } from '@/components/auth.tsx';

const getSession = vi.hoisted(() => vi.fn());
vi.mock('@/api/auth.ts', () => ({ getSession }));

describe('ProtectedRoute', () => {
  it('waits for the server session before rendering protected content', async () => {
    let resolveSession: (value: unknown) => void = () => undefined;
    getSession.mockReturnValue(
      new Promise((resolve) => {
        resolveSession = resolve;
      })
    );

    renderRoute();
    expect(screen.getByRole('status')).toBeInTheDocument();
    expect(screen.queryByText('protected')).not.toBeInTheDocument();

    resolveSession({
      code: 0,
      data: { authenticated: true, username: 'user', authSource: 'oidc', admin: true }
    });
    expect(await screen.findByText('protected')).toBeInTheDocument();
  });

  it('redirects an unauthenticated session to login', async () => {
    getSession.mockResolvedValue({ code: 0, data: { authenticated: false } });
    renderRoute();
    expect(await screen.findByText('login')).toBeInTheDocument();
  });
});

function renderRoute() {
  render(
    <MemoryRouter initialEntries={['/']}>
      <Routes>
        <Route path="/" element={<ProtectedRoute>protected</ProtectedRoute>} />
        <Route path="/auth/login" element={<>login</>} />
      </Routes>
    </MemoryRouter>
  );
}
