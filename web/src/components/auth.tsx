import { ReactNode, useEffect, useState } from 'react';
import { Spin } from 'antd';
import { Navigate } from 'react-router-dom';

import { getSession } from '@/api/auth.ts';

export const ProtectedRoute = ({ children }: { children: ReactNode }) => {
  const [authenticated, setAuthenticated] = useState<boolean | null>(null);

  useEffect(() => {
    let active = true;

    getSession()
      .then((rsp) => {
        if (active) {
          setAuthenticated(rsp.code === 0 && rsp.data.authenticated);
        }
      })
      .catch(() => {
        if (active) setAuthenticated(false);
      });

    return () => {
      active = false;
    };
  }, []);

  if (authenticated === null) {
    return (
      <div className="flex h-screen w-screen items-center justify-center" role="status">
        <Spin size="large" />
      </div>
    );
  }

  if (!authenticated) {
    return <Navigate to={'/auth/login'} replace />;
  }

  return children;
};
