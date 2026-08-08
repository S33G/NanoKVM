import type { MockScenarioName, MockState } from './types';

const now = '2026-08-08T10:30:00Z';

function baseScenario(scenario: MockScenarioName): MockState {
  return {
    application: {
      preview: false,
      updateServer: { enabled: false, url: 'https://github.com/sipeed/NanoKVM/releases' },
      version: '2.3.1'
    },
    auth: {
      allowLocalLogin: true,
      oidcEnabled: false,
      oidcReady: false,
      passwordUpdated: true,
      providerName: 'OpenID Connect',
      session: {
        admin: true,
        authenticated: true,
        authSource: 'local',
        displayName: 'NanoKVM Developer',
        username: 'admin'
      }
    },
    delay: 80,
    device: {
      hardware: 'pcie',
      hostname: 'nanokvm-mock',
      imageVersion: '2026-07-24',
      ip: '192.168.1.86',
      mdns: true,
      memoryLimit: { enabled: true, limit: 256 },
      mouseJiggler: { enabled: false, mode: 'absolute' },
      oledSleep: 60,
      ssh: true,
      swap: 512,
      title: 'NanoKVM Mock Lab',
      version: '2.3.1'
    },
    faults: {},
    hid: {
      leaderKey: 'ControlRight',
      mode: 'absolute',
      shortcuts: [
        { id: 'shortcut-1', keys: ['ControlLeft', 'AltLeft', 'Delete'] },
        { id: 'shortcut-2', keys: ['MetaLeft', 'KeyL'] }
      ]
    },
    network: {
      dns: { mode: 'dhcp', servers: ['192.168.1.1'] },
      wifi: { connected: true, ip: '192.168.1.86', ssid: 'NanoKVM Lab' },
      wol: [{ mac: '00:1A:2B:3C:4D:5E', name: 'Build server' }]
    },
    picoclaw: {
      installed: true,
      modelConfigured: true,
      modelName: 'mock-gpt',
      running: true,
      sessions: [
        {
          created: now,
          id: 'mock-session-1',
          messages: [
            { role: 'user', content: 'Check the attached host.' },
            { role: 'assistant', content: 'The mock host is online and showing its desktop.' }
          ],
          title: 'Host status check',
          updated: now
        }
      ]
    },
    scenario,
    scripts: {
      autostart: { 'welcome.sh': '#!/bin/sh\necho "NanoKVM mock ready"\n' },
      files: ['reboot-host.sh', 'type-password.js']
    },
    services: {
      hdmi: true,
      mcp: { apiKey: 'mcp_mock_7f4c2d', controlMode: 'off', enabled: false },
      tailscale: {
        installed: true,
        ip: '100.86.42.10',
        loggedIn: true,
        running: true
      }
    },
    storage: {
      cdrom: true,
      downloadProgress: 100,
      images: ['ubuntu-24.04-live-server.iso', 'rescue-system.iso'],
      mounted: 'ubuntu-24.04-live-server.iso'
    },
    virtualDevice: 'network',
  };
}

export function createScenario(name: MockScenarioName): MockState {
  const state = baseScenario(name);

  if (name === 'auth-local') {
    state.auth.session.authenticated = false;
  }

  if (name === 'auth-oidc') {
    state.auth.oidcEnabled = true;
    state.auth.oidcReady = true;
    state.auth.session = {
      admin: true,
      authenticated: false,
      authSource: 'oidc',
      username: ''
    };
  }

  if (name === 'lite-degraded') {
    state.device.hardware = 'beta';
    state.device.imageVersion = '2024-01-10';
    state.services.hdmi = false;
    state.services.tailscale = { installed: false, ip: '', loggedIn: false, running: false };
    state.picoclaw = {
      installed: false,
      modelConfigured: false,
      modelName: '',
      running: false,
      sessions: []
    };
    state.storage.images = [];
    state.storage.mounted = '';
    state.virtualDevice = 'none';
  }

  if (name === 'settings-lifecycle') {
    state.device.hostname = 'nanokvm-settings-test';
    state.application.preview = true;
    state.services.mcp.enabled = true;
    state.services.mcp.controlMode = 'mcp';
  }

  if (name === 'picoclaw-lifecycle') {
    state.picoclaw.installed = false;
    state.picoclaw.modelConfigured = false;
    state.picoclaw.modelName = '';
    state.picoclaw.running = false;
    state.picoclaw.sessions = [];
  }

  return state;
}
