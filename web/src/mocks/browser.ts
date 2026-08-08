import { setupWorker } from 'msw/browser';

import { restHandlers } from './handlers';
import { websocketHandlers } from './websockets';

export const handlers = [...restHandlers, ...websocketHandlers];
export const worker = setupWorker(...handlers);

export async function startMockWorker() {
  return worker.start({
    onUnhandledRequest(request, print) {
      const pathname = new URL(request.url).pathname;
      if (pathname.startsWith('/api/')) {
        throw new Error(`[MSW] Unexpected API request: ${request.method} ${pathname}`);
      }
      print.warning();
    }
  });
}
