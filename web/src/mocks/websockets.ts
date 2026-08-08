import { ws } from 'msw';

import { getMockState, updateMockState } from './state';

const hid = ws.link('/api/ws');
const terminal = ws.link('/api/vm/terminal');
const picoclaw = ws.link('/api/picoclaw/gateway/ws');

const hidHandler = hid.addEventListener('connection', ({ client }) => {
  client.addEventListener('message', () => {
    // Heartbeats and binary HID reports are intentionally accepted without echoing them.
  });
});

const terminalHandler = terminal.addEventListener('connection', ({ client }) => {
  client.send('\r\nNanoKVM mock shell\r\nroot@nanokvm-mock:~# ');
  client.addEventListener('message', async (event) => {
    let input = '';
    if (typeof event.data === 'string') input = event.data;
    else if (event.data instanceof Blob) input = await event.data.text();
    else if (event.data instanceof ArrayBuffer) input = new TextDecoder().decode(event.data);

    if (!input || input.startsWith('{')) return;
    client.send(input);
    if (input.includes('\r') || input.includes('\n')) {
      client.send('NanoKVM mocked command completed\r\nroot@nanokvm-mock:~# ');
    }
  });
});

const picoclawHandler = picoclaw.addEventListener('connection', ({ client }) => {
  client.addEventListener('message', (event) => {
    if (typeof event.data !== 'string') return;

    let message: Record<string, unknown>;
    try {
      message = JSON.parse(event.data) as Record<string, unknown>;
    } catch {
      return;
    }

    const type = String(message.type || '');
    if (type === 'ping') {
      client.send(JSON.stringify({ type: 'pong' }));
      return;
    }
    if (type === 'message.cancel') {
      client.send(JSON.stringify({ type: 'typing.stop' }));
      return;
    }
    if (type !== 'message.send') return;

    const payload = (message.payload || {}) as Record<string, unknown>;
    const content = String(payload.content || '');
    const id = String(message.id || crypto.randomUUID());
    const sessionId = String(message.session_id || new URL(client.url).searchParams.get('session_id') || 'mock-session');
    const response = `Mock PicoClaw received: ${content}`;

    client.send(JSON.stringify({ type: 'typing.start', id, session_id: sessionId }));
    window.setTimeout(() => {
      const now = new Date().toISOString();
      updateMockState((state) => {
        let session = state.picoclaw.sessions.find((item) => item.id === sessionId);
        if (!session) {
          session = {
            id: sessionId,
            title: content.slice(0, 48) || 'Mock conversation',
            messages: [],
            created: now,
            updated: now
          };
          state.picoclaw.sessions.unshift(session);
        }
        session.messages.push({ role: 'user', content }, { role: 'assistant', content: response });
        session.updated = now;
      });
      if (!getMockState().picoclaw.running) return;
      client.send(JSON.stringify({ type: 'typing.stop', id, session_id: sessionId }));
      client.send(
        JSON.stringify({
          type: 'message.create',
          id: `assistant-${id}`,
          session_id: sessionId,
          payload: { role: 'assistant', content: response }
        })
      );
    }, Math.max(100, getMockState().delay));
  });
});

export const websocketHandlers = [hidHandler, terminalHandler, picoclawHandler];
