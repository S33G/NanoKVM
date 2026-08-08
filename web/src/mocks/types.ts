export const mockScenarioNames = [
  'auth-local',
  'auth-oidc',
  'pcie-full',
  'lite-degraded',
  'settings-lifecycle',
  'picoclaw-lifecycle'
] as const;

export type MockScenarioName = (typeof mockScenarioNames)[number];

export type MockFault = {
  delay: number;
  enabled: boolean;
  message: string;
  once: boolean;
  status: number;
};

export type MockSession = {
  admin: boolean;
  authenticated: boolean;
  authSource: 'local' | 'oidc';
  displayName?: string;
  email?: string;
  username: string;
};

export type MockState = {
  application: {
    preview: boolean;
    updateServer: { enabled: boolean; url: string };
    version: string;
  };
  auth: {
    allowLocalLogin: boolean;
    oidcEnabled: boolean;
    oidcReady: boolean;
    passwordUpdated: boolean;
    providerName: string;
    session: MockSession;
  };
  delay: number;
  device: {
    hardware: 'beta' | 'pcie' | 'alpha';
    hostname: string;
    imageVersion: string;
    ip: string;
    mdns: boolean;
    memoryLimit: { enabled: boolean; limit: number };
    mouseJiggler: { enabled: boolean; mode: string };
    oledSleep: number;
    ssh: boolean;
    swap: number;
    title: string;
    version: string;
  };
  faults: Record<string, MockFault>;
  hid: {
    leaderKey: string;
    mode: string;
    shortcuts: Array<{ id: string; keys: unknown[] }>;
  };
  network: {
    dns: { mode: 'dhcp' | 'manual'; servers: string[] };
    wifi: { connected: boolean; ip: string; ssid: string };
    wol: Array<{ mac: string; name: string }>;
  };
  scenario: MockScenarioName;
  scripts: {
    autostart: Record<string, string>;
    files: string[];
  };
  services: {
    hdmi: boolean;
    mcp: { apiKey: string; controlMode: 'off' | 'mcp' | 'picoclaw'; enabled: boolean };
    tailscale: { installed: boolean; ip: string; loggedIn: boolean; running: boolean };
  };
  storage: {
    cdrom: boolean;
    downloadProgress: number;
    images: string[];
    mounted: string;
  };
  virtualDevice: string;
  picoclaw: {
    installed: boolean;
    modelConfigured: boolean;
    modelName: string;
    running: boolean;
    sessions: Array<{
      created: string;
      id: string;
      messages: Array<{ content: string; role: 'user' | 'assistant' }>;
      title: string;
      updated: string;
    }>;
  };
};
