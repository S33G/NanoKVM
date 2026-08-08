import { useEffect } from 'react';
import { useAtom } from 'jotai';
import { Helmet, HelmetData } from 'react-helmet-async';

import { getSession } from '@/api/auth.ts';
import { getWebTitle } from '@/api/vm.ts';
import { webTitleAtom } from '@/jotai/settings.ts';

type HeadProps = {
  title?: string;
  description?: string;
};

const helmetData = new HelmetData({});

export const Head = ({ title = '', description = '' }: HeadProps = {}) => {
  const [webTitle, setWebTitle] = useAtom(webTitleAtom);

  useEffect(() => {
    getSession()
      .then((session) => {
        if (!session.data.authenticated) return;

        return getWebTitle().then((rsp) => {
          if (rsp.data?.title) {
            setWebTitle(rsp.data.title);
          }
        });
      })
      .catch(() => undefined);
  }, [setWebTitle]);

  return (
    <Helmet
      helmetData={helmetData}
      title={webTitle ? webTitle : title ? `${title} - NanoKVM` : undefined}
      defaultTitle={webTitle}
    >
      <meta name="description" content={description} />
    </Helmet>
  );
};
