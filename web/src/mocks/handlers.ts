import { delay, http, HttpResponse, type HttpResponseResolver } from 'msw';

import { consumeMockFault, getMockState, updateMockState } from './state';

type JsonObject = Record<string, unknown>;

let activeDownload = '';

function success(data: unknown = null) {
  return HttpResponse.json({ code: 0, data });
}

function endpoint(resolver: HttpResponseResolver): HttpResponseResolver {
  return async (info) => {
    const pathname = new URL(info.request.url).pathname;
    const fault = consumeMockFault(pathname);
    await delay(getMockState().delay + (fault?.delay ?? 0));
    if (fault) {
      return HttpResponse.json(
        { code: fault.status, data: null, msg: fault.message },
        { status: fault.status }
      );
    }
    return resolver(info);
  };
}

async function json(request: Request) {
  return (await request.json()) as JsonObject;
}

function runtimeStatus() {
  const state = getMockState();
  const mode = state.services.mcp.controlMode;
  return {
    ready: state.picoclaw.installed && state.picoclaw.modelConfigured && state.picoclaw.running,
    installed: state.picoclaw.installed,
    installing: false,
    install_progress: state.picoclaw.installed ? 100 : 0,
    install_stage: state.picoclaw.installed ? 'complete' : 'not_installed',
    agent_profile: 'default',
    model_configured: state.picoclaw.modelConfigured,
    model_name: state.picoclaw.modelName,
    status: state.picoclaw.running ? 'ready' : state.picoclaw.installed ? 'stopped' : 'not_installed',
    checked_at: new Date().toISOString(),
    current_session: '',
    runtime_intent: { desired_running: state.picoclaw.running },
    control_mode: mode,
    transitioning: false,
    control: { mode, transitioning: false, can_control: mode === 'picoclaw' },
    capabilities: {
      chat: state.picoclaw.running && state.picoclaw.modelConfigured,
      read_only_tools: state.picoclaw.running,
      device_write: mode === 'picoclaw'
    }
  };
}

function mcpConfig() {
  const config = getMockState().services.mcp;
  return { ...config, transitioning: false };
}

function tailscaleStatus() {
  const tailscale = getMockState().services.tailscale;
  const state = !tailscale.installed
    ? 'notInstall'
    : !tailscale.running
      ? 'notRunning'
      : !tailscale.loggedIn
        ? 'notLogin'
        : 'running';
  return {
    state,
    name: tailscale.loggedIn ? 'nanokvm-mock.tailnet.ts.net' : '',
    ip: tailscale.ip,
    account: tailscale.loggedIn ? 'developer@example.com' : ''
  };
}

function newKey() {
  return `mcp_mock_${crypto.randomUUID().replace(/-/g, '').slice(0, 12)}`;
}

export const restHandlers = [
  // Authentication
  http.get('/api/auth/config', endpoint(() => success(getMockState().auth))),
  http.get('/api/auth/session', endpoint(() => success(getMockState().auth.session))),
  http.post(
    '/api/auth/login',
    endpoint(async ({ request }) => {
      const body = await json(request);
      updateMockState((state) => {
        state.auth.session.authenticated = true;
        state.auth.session.authSource = 'local';
        state.auth.session.username = String(body.username || 'admin');
        state.auth.session.admin = true;
      });
      return success({ token: 'mocked_token' });
    })
  ),
  http.post(
    '/api/auth/logout',
    endpoint(() => {
      updateMockState((state) => {
        state.auth.session.authenticated = false;
      });
      return success();
    })
  ),
  http.get(
    '/api/auth/account',
    endpoint(() => success({ username: getMockState().auth.session.username }))
  ),
  http.get(
    '/api/auth/password',
    endpoint(() => success({ isUpdated: getMockState().auth.passwordUpdated }))
  ),
  http.post(
    '/api/auth/password',
    endpoint(async ({ request }) => {
      const body = await json(request);
      updateMockState((state) => {
        state.auth.passwordUpdated = true;
        state.auth.session.username = String(body.username || state.auth.session.username);
      });
      return success();
    })
  ),

  // Application updates
  http.get(
    '/api/application/version',
    endpoint(() => {
      const version = getMockState().application.version;
      return success({ current: version, latest: version });
    })
  ),
  http.post('/api/application/update', endpoint(() => success())),
  http.post('/api/application/update/offline', endpoint(() => success())),
  http.get(
    '/api/application/preview',
    endpoint(() => success({ enabled: getMockState().application.preview }))
  ),
  http.post(
    '/api/application/preview',
    endpoint(async ({ request }) => {
      const body = await json(request);
      updateMockState((state) => {
        state.application.preview = body.enable === true;
      });
      return success({ enabled: getMockState().application.preview });
    })
  ),
  http.get(
    '/api/application/update-server',
    endpoint(() => success(getMockState().application.updateServer))
  ),
  http.post(
    '/api/application/update-server',
    endpoint(async ({ request }) => {
      const body = await json(request);
      updateMockState((state) => {
        state.application.updateServer = {
          enabled: body.enabled === true,
          url: String(body.url || '')
        };
      });
      return success(getMockState().application.updateServer);
    })
  ),

  // VM and device settings
  http.get(
    '/api/vm/info',
    endpoint(() => {
      const state = getMockState();
      return success({
        ips: [{ name: 'eth0', addr: state.device.ip, version: 'IPv4', type: 'Ethernet' }],
        mdns: state.device.mdns ? `${state.device.hostname}.local` : '',
        image: state.device.imageVersion,
        application: state.device.version,
        deviceKey: 'MOCK-NANOKVM'
      });
    })
  ),
  http.get(
    '/api/vm/hardware',
    endpoint(() => success({ version: getMockState().device.hardware.toUpperCase() }))
  ),
  http.get('/api/vm/gpio', endpoint(() => success({ pwr: true, hdd: false }))),
  http.post('/api/vm/gpio', endpoint(() => success())),
  http.post('/api/vm/screen', endpoint(() => success())),
  http.get('/api/vm/memory/limit', endpoint(() => success(getMockState().device.memoryLimit))),
  http.post(
    '/api/vm/memory/limit',
    endpoint(async ({ request }) => {
      const body = await json(request);
      updateMockState((state) => {
        state.device.memoryLimit = { enabled: body.enabled === true, limit: Number(body.limit) };
      });
      return success(getMockState().device.memoryLimit);
    })
  ),
  http.get(
    '/api/vm/oled',
    endpoint(() => success({ exist: true, sleep: getMockState().device.oledSleep }))
  ),
  http.post(
    '/api/vm/oled',
    endpoint(async ({ request }) => {
      const body = await json(request);
      updateMockState((state) => {
        state.device.oledSleep = Number(body.sleep);
      });
      return success();
    })
  ),
  http.get(
    '/api/vm/hdmi',
    endpoint(() => success({ enabled: getMockState().services.hdmi, idleTimeout: 0 }))
  ),
  http.post('/api/vm/hdmi/reset', endpoint(() => success())),
  http.post(
    '/api/vm/hdmi/enable',
    endpoint(() => {
      updateMockState((state) => {
        state.services.hdmi = true;
      });
      return success();
    })
  ),
  http.post(
    '/api/vm/hdmi/disable',
    endpoint(() => {
      updateMockState((state) => {
        state.services.hdmi = false;
      });
      return success();
    })
  ),
  http.post('/api/vm/hdmi/timeout', endpoint(() => success())),
  http.get('/api/vm/ssh', endpoint(() => success({ enabled: getMockState().device.ssh }))),
  http.post(
    '/api/vm/ssh/enable',
    endpoint(() => {
      updateMockState((state) => {
        state.device.ssh = true;
      });
      return success();
    })
  ),
  http.post(
    '/api/vm/ssh/disable',
    endpoint(() => {
      updateMockState((state) => {
        state.device.ssh = false;
      });
      return success();
    })
  ),
  http.get('/api/vm/swap', endpoint(() => success({ size: getMockState().device.swap }))),
  http.post(
    '/api/vm/swap',
    endpoint(async ({ request }) => {
      const body = await json(request);
      updateMockState((state) => {
        state.device.swap = Number(body.size);
      });
      return success();
    })
  ),
  http.get('/api/vm/mouse-jiggler', endpoint(() => success(getMockState().device.mouseJiggler))),
  http.post(
    '/api/vm/mouse-jiggler',
    endpoint(async ({ request }) => {
      const body = await json(request);
      updateMockState((state) => {
        state.device.mouseJiggler = {
          enabled: body.enabled === true,
          mode: String(body.mode || 'relative')
        };
      });
      return success();
    })
  ),
  http.get('/api/vm/hostname', endpoint(() => success({ hostname: getMockState().device.hostname }))),
  http.post(
    '/api/vm/hostname',
    endpoint(async ({ request }) => {
      const body = await json(request);
      updateMockState((state) => {
        state.device.hostname = String(body.hostname || '');
      });
      return success();
    })
  ),
  http.get('/api/vm/web-title', endpoint(() => success({ title: getMockState().device.title }))),
  http.post(
    '/api/vm/web-title',
    endpoint(async ({ request }) => {
      const body = await json(request);
      updateMockState((state) => {
        state.device.title = String(body.title || '');
      });
      return success();
    })
  ),
  http.get('/api/vm/mdns', endpoint(() => success({ enabled: getMockState().device.mdns }))),
  ...(['enable', 'disable'] as const).map((action) =>
    http.post(
      `/api/vm/mdns/${action}`,
      endpoint(() => {
        updateMockState((state) => {
          state.device.mdns = action === 'enable';
        });
        return success();
      })
    )
  ),
  http.post('/api/vm/tls', endpoint(() => success())),
  http.post('/api/vm/system/reboot', endpoint(() => success())),
  http.get(
    '/api/vm/device/virtual',
    endpoint(() => {
      const device = getMockState().virtualDevice;
      return success({ disk: device === 'disk', network: device === 'network' });
    })
  ),
  http.post(
    '/api/vm/device/virtual',
    endpoint(async ({ request }) => {
      const body = await json(request);
      const requested = String(body.device || 'none');
      updateMockState((state) => {
        state.virtualDevice = state.virtualDevice === requested ? 'none' : requested;
      });
      return success();
    })
  ),

  // Autostart and uploaded scripts
  http.get(
    '/api/vm/autostart',
    endpoint(() => success({ files: Object.keys(getMockState().scripts.autostart) }))
  ),
  http.get(
    '/api/vm/autostart/:name',
    endpoint(({ params }) => success(getMockState().scripts.autostart[String(params.name)] || ''))
  ),
  http.post(
    '/api/vm/autostart/:name',
    endpoint(async ({ request, params }) => {
      const body = await json(request);
      updateMockState((state) => {
        state.scripts.autostart[String(params.name)] = String(body.content || '');
      });
      return success();
    })
  ),
  http.delete(
    '/api/vm/autostart/:name',
    endpoint(({ params }) => {
      updateMockState((state) => {
        delete state.scripts.autostart[String(params.name)];
      });
      return success();
    })
  ),
  http.get('/api/vm/script', endpoint(() => success({ files: getMockState().scripts.files }))),
  http.post(
    '/api/vm/script/upload',
    endpoint(async ({ request }) => {
      const form = await request.formData();
      const file = form.get('file');
      const name = file instanceof File ? file.name : 'uploaded-script.sh';
      updateMockState((state) => {
        if (!state.scripts.files.includes(name)) state.scripts.files.push(name);
      });
      return success({ file: name });
    })
  ),
  http.post(
    '/api/vm/script/run',
    endpoint(async ({ request }) => {
      const body = await json(request);
      return success({ log: `NanoKVM mock executed ${String(body.name || 'script')}\nexit 0` });
    })
  ),
  http.delete(
    '/api/vm/script',
    endpoint(async ({ request }) => {
      const body = await json(request);
      updateMockState((state) => {
        state.scripts.files = state.scripts.files.filter((file) => file !== body.name);
      });
      return success();
    })
  ),

  // HID
  http.post('/api/hid/paste', endpoint(() => success())),
  http.post('/api/hid/reset', endpoint(() => success())),
  http.get('/api/hid/mode', endpoint(() => success({ mode: getMockState().hid.mode }))),
  http.post(
    '/api/hid/mode',
    endpoint(async ({ request }) => {
      const body = await json(request);
      updateMockState((state) => {
        state.hid.mode = String(body.mode || 'absolute');
      });
      return success();
    })
  ),
  http.get(
    '/api/hid/leds',
    endpoint(() =>
      success({
        numLock: false,
        capsLock: false,
        scrollLock: false,
        known: true,
        updatedAt: new Date().toISOString()
      })
    )
  ),
  http.get('/api/hid/shortcuts', endpoint(() => success({ shortcuts: getMockState().hid.shortcuts }))),
  http.post(
    '/api/hid/shortcut',
    endpoint(async ({ request }) => {
      const body = await json(request);
      updateMockState((state) => {
        state.hid.shortcuts.push({ id: crypto.randomUUID(), keys: (body.keys as unknown[]) || [] });
      });
      return success();
    })
  ),
  http.delete(
    '/api/hid/shortcut',
    endpoint(async ({ request }) => {
      const body = await json(request);
      updateMockState((state) => {
        state.hid.shortcuts = state.hid.shortcuts.filter((shortcut) => shortcut.id !== body.id);
      });
      return success();
    })
  ),
  http.get(
    '/api/hid/shortcut/leader-key',
    endpoint(() => success({ key: getMockState().hid.leaderKey }))
  ),
  http.post(
    '/api/hid/shortcut/leader-key',
    endpoint(async ({ request }) => {
      const body = await json(request);
      updateMockState((state) => {
        state.hid.leaderKey = String(body.key || 'ControlRight');
      });
      return success();
    })
  ),

  // Storage and downloads
  http.get('/api/storage/image', endpoint(() => success({ files: getMockState().storage.images }))),
  http.get(
    '/api/storage/image/mounted',
    endpoint(() => success({ file: getMockState().storage.mounted }))
  ),
  http.get(
    '/api/storage/cdrom',
    endpoint(() => success({ cdrom: getMockState().storage.cdrom ? 1 : 0 }))
  ),
  http.post(
    '/api/storage/image/mount',
    endpoint(async ({ request }) => {
      const body = await json(request);
      updateMockState((state) => {
        state.storage.mounted = String(body.file || '');
        state.storage.cdrom = body.cdrom === true;
      });
      return success();
    })
  ),
  http.post(
    '/api/storage/image/delete',
    endpoint(async ({ request }) => {
      const body = await json(request);
      updateMockState((state) => {
        state.storage.images = state.storage.images.filter((file) => file !== body.file);
      });
      return success();
    })
  ),
  http.get('/api/download/image/enabled', endpoint(() => success({ enabled: true }))),
  http.post(
    '/api/download/image',
    endpoint(async ({ request }) => {
      const body = await json(request);
      activeDownload = String(body.file || 'mock-download.iso');
      updateMockState((state) => {
        state.storage.downloadProgress = 1;
      });
      return success();
    })
  ),
  http.post(
    '/api/download/image/cancel',
    endpoint(() => {
      activeDownload = '';
      updateMockState((state) => {
        state.storage.downloadProgress = 100;
      });
      return success();
    })
  ),
  http.get(
    '/api/download/image/status',
    endpoint(() => {
      if (!activeDownload) return success({ status: 'idle', file: '', percentage: '' });
      let progress = getMockState().storage.downloadProgress;
      progress = Math.min(100, progress + 40);
      updateMockState((state) => {
        state.storage.downloadProgress = progress;
      });
      if (progress >= 100) {
        const completed = activeDownload;
        const filename = completed.split('/').pop() || 'download.iso';
        activeDownload = '';
        updateMockState((state) => {
          if (!state.storage.images.includes(filename)) state.storage.images.push(filename);
        });
        return success({ status: 'success', file: completed, percentage: '100%' });
      }
      return success({ status: 'in_progress', file: activeDownload, percentage: `${progress}%` });
    })
  ),
  http.post(
    '/api/download/file',
    endpoint(async ({ request }) => {
      const form = await request.formData();
      const file = form.get('file');
      const name = file instanceof File ? file.name : 'uploaded.iso';
      updateMockState((state) => {
        if (!state.storage.images.includes(name)) state.storage.images.push(name);
        state.storage.downloadProgress = 100;
      });
      return success();
    })
  ),

  // Network
  http.get(
    '/api/network/wol/mac',
    endpoint(() =>
      success({ macs: getMockState().network.wol.map(({ mac, name }) => `${mac} ${name}`.trim()) })
    )
  ),
  http.post(
    '/api/network/wol',
    endpoint(async ({ request }) => {
      const body = await json(request);
      const mac = String(body.mac || '');
      updateMockState((state) => {
        if (mac && !state.network.wol.some((item) => item.mac === mac)) {
          state.network.wol.push({ mac, name: '' });
        }
      });
      return success();
    })
  ),
  http.delete(
    '/api/network/wol/mac',
    endpoint(async ({ request }) => {
      const body = await json(request);
      updateMockState((state) => {
        state.network.wol = state.network.wol.filter((item) => item.mac !== body.mac);
      });
      return success();
    })
  ),
  http.post(
    '/api/network/wol/mac/name',
    endpoint(async ({ request }) => {
      const body = await json(request);
      updateMockState((state) => {
        const entry = state.network.wol.find((item) => item.mac === body.mac);
        if (entry) entry.name = String(body.name || '');
      });
      return success();
    })
  ),
  http.get('/api/network/wifi', endpoint(() => success(getMockState().network.wifi))),
  ...(['/api/network/wifi', '/api/network/wifi/connect'] as const).map((path) =>
    http.post(
      path,
      endpoint(async ({ request }) => {
        const body = await json(request);
        updateMockState((state) => {
          state.network.wifi = {
            connected: true,
            ip: state.device.ip,
            ssid: String(body.ssid || '')
          };
        });
        return success();
      })
    )
  ),
  http.post('/api/network/wifi/verify', endpoint(() => success())),
  http.post(
    '/api/network/wifi/disconnect',
    endpoint(() => {
      updateMockState((state) => {
        state.network.wifi = { connected: false, ip: '', ssid: '' };
      });
      return success();
    })
  ),
  http.get(
    '/api/network/dns',
    endpoint(() => {
      const dns = getMockState().network.dns;
      return success({ ...dns, dhcp: ['192.168.1.1'], info: { interface: 'eth0' } });
    })
  ),
  http.post(
    '/api/network/dns',
    endpoint(async ({ request }) => {
      const body = await json(request);
      updateMockState((state) => {
        state.network.dns = {
          mode: body.mode === 'manual' ? 'manual' : 'dhcp',
          servers: Array.isArray(body.servers) ? body.servers.map(String) : []
        };
      });
      return success();
    })
  ),

  // Stream controls
  http.get(
    '/api/stream/mjpeg',
    endpoint(() =>
      new HttpResponse(
        `<svg xmlns="http://www.w3.org/2000/svg" width="1280" height="720" viewBox="0 0 1280 720">
          <defs><linearGradient id="bg" x2="1" y2="1"><stop stop-color="#111827"/><stop offset="1" stop-color="#020617"/></linearGradient></defs>
          <rect width="1280" height="720" fill="url(#bg)"/><circle cx="640" cy="310" r="72" fill="#2563eb"/>
          <text x="640" y="430" fill="#e5e7eb" font-family="monospace" font-size="38" text-anchor="middle">NanoKVM Mock Display</text>
          <text x="640" y="480" fill="#64748b" font-family="monospace" font-size="22" text-anchor="middle">${getMockState().device.hostname}</text>
        </svg>`,
        { headers: { 'Content-Type': 'image/svg+xml', 'Cache-Control': 'no-store' } }
      )
    )
  ),
  http.post('/api/stream/mjpeg/detect', endpoint(() => success())),
  http.post('/api/stream/mjpeg/detect/stop', endpoint(() => success())),

  // Tailscale
  http.get('/api/extensions/tailscale/status', endpoint(() => success(tailscaleStatus()))),
  http.post(
    '/api/extensions/tailscale/install',
    endpoint(() => {
      updateMockState((state) => {
        state.services.tailscale.installed = true;
        state.services.tailscale.running = false;
      });
      return success();
    })
  ),
  http.post(
    '/api/extensions/tailscale/uninstall',
    endpoint(() => {
      updateMockState((state) => {
        state.services.tailscale = { installed: false, running: false, loggedIn: false, ip: '' };
      });
      return success();
    })
  ),
  ...(['start', 'restart'] as const).map((action) =>
    http.post(
      `/api/extensions/tailscale/${action}`,
      endpoint(() => {
        updateMockState((state) => {
          state.services.tailscale.running = true;
        });
        return success();
      })
    )
  ),
  http.post(
    '/api/extensions/tailscale/stop',
    endpoint(() => {
      updateMockState((state) => {
        state.services.tailscale.running = false;
      });
      return success();
    })
  ),
  http.post(
    '/api/extensions/tailscale/up',
    endpoint(() => {
      updateMockState((state) => {
        state.services.tailscale.loggedIn = true;
      });
      return success();
    })
  ),
  http.post(
    '/api/extensions/tailscale/down',
    endpoint(() => {
      updateMockState((state) => {
        state.services.tailscale.loggedIn = false;
      });
      return success();
    })
  ),
  http.post(
    '/api/extensions/tailscale/login',
    endpoint(() => {
      updateMockState((state) => {
        state.services.tailscale.loggedIn = true;
        state.services.tailscale.running = true;
        if (!state.services.tailscale.ip) state.services.tailscale.ip = '100.86.42.10';
      });
      return success({ url: '' });
    })
  ),
  http.post(
    '/api/extensions/tailscale/logout',
    endpoint(() => {
      updateMockState((state) => {
        state.services.tailscale.loggedIn = false;
      });
      return success();
    })
  ),

  // MCP and PicoClaw
  http.get('/api/mcp/config', endpoint(() => success(mcpConfig()))),
  http.post(
    '/api/mcp/config',
    endpoint(async ({ request }) => {
      const body = await json(request);
      updateMockState((state) => {
        state.services.mcp.enabled = body.enabled === true;
        state.services.mcp.controlMode = body.enabled === true ? 'mcp' : 'off';
      });
      return success(mcpConfig());
    })
  ),
  http.post(
    '/api/mcp/key/regenerate',
    endpoint(() => {
      updateMockState((state) => {
        state.services.mcp.apiKey = newKey();
      });
      return success(mcpConfig());
    })
  ),
  http.get('/api/picoclaw/runtime/status', endpoint(() => success(runtimeStatus()))),
  http.post(
    '/api/picoclaw/runtime/install',
    endpoint(() => {
      updateMockState((state) => {
        state.picoclaw.installed = true;
      });
      return success({ installed: true, binary: '/usr/bin/picoclaw', download: 'mock', status: runtimeStatus() });
    })
  ),
  http.post(
    '/api/picoclaw/runtime/uninstall',
    endpoint(() => {
      updateMockState((state) => {
        state.picoclaw.installed = false;
        state.picoclaw.running = false;
        state.picoclaw.modelConfigured = false;
        state.services.mcp.controlMode = 'off';
      });
      return success({ status: runtimeStatus() });
    })
  ),
  ...(['start', 'stop'] as const).map((action) =>
    http.post(
      `/api/picoclaw/runtime/${action}`,
      endpoint(() => {
        updateMockState((state) => {
          state.picoclaw.running = action === 'start';
          state.services.mcp.controlMode = action === 'start' ? 'picoclaw' : 'off';
        });
        return success({ started: action === 'start', command: `picoclaw ${action}`, status: runtimeStatus() });
      })
    )
  ),
  http.delete('/api/picoclaw/runtime/session', endpoint(() => success())),
  http.post(
    '/api/picoclaw/model/config',
    endpoint(async ({ request }) => {
      const body = await json(request);
      updateMockState((state) => {
        state.picoclaw.modelConfigured = true;
        state.picoclaw.modelName = String(body.model || 'mock-gpt');
      });
      return success({ status: runtimeStatus() });
    })
  ),
  http.post('/api/picoclaw/agent/profile', endpoint(() => success({ status: runtimeStatus() }))),
  http.get(
    '/api/picoclaw/sessions',
    endpoint(({ request }) => {
      const url = new URL(request.url);
      const offset = Number(url.searchParams.get('offset') || 0);
      const limit = Number(url.searchParams.get('limit') || 100);
      return success(
        getMockState()
          .picoclaw.sessions.slice(offset, offset + limit)
          .map((session) => ({
            id: session.id,
            title: session.title,
            preview: session.messages[session.messages.length - 1]?.content || '',
            message_count: session.messages.length,
            created: session.created,
            updated: session.updated
          }))
      );
    })
  ),
  http.get(
    '/api/picoclaw/sessions/:id',
    endpoint(({ params }) => {
      const session = getMockState().picoclaw.sessions.find((item) => item.id === params.id);
      return session
        ? success(session)
        : HttpResponse.json({ code: 404, data: null, msg: 'PicoClaw session not found' }, { status: 404 });
    })
  ),
  http.delete(
    '/api/picoclaw/sessions/:id',
    endpoint(({ params }) => {
      updateMockState((state) => {
        state.picoclaw.sessions = state.picoclaw.sessions.filter((item) => item.id !== params.id);
      });
      return success();
    })
  ),
  http.get(
    '/api/ai/control/status',
    endpoint(() => {
      const mode = getMockState().services.mcp.controlMode;
      return success({ mode, transitioning: false, can_control: mode === 'picoclaw' });
    })
  ),
  http.put(
    '/api/ai/control/mode',
    endpoint(async ({ request }) => {
      const body = await json(request);
      const mode = body.mode === 'mcp' || body.mode === 'picoclaw' ? body.mode : 'off';
      updateMockState((state) => {
        state.services.mcp.controlMode = mode;
      });
      return success({ mode, transitioning: false, can_control: mode === 'picoclaw', status: runtimeStatus() });
    })
  ),

  http.all(
    '/api/*',
    endpoint(({ request }) => {
      const pathname = new URL(request.url).pathname;
      return HttpResponse.json(
        { code: 501, data: null, msg: `No mocked API handler for ${request.method} ${pathname}` },
        { status: 501 }
      );
    })
  )
];
