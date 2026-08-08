import { useState, useSyncExternalStore } from 'react';
import { ApiOutlined, ReloadOutlined } from '@ant-design/icons';
import { Button, Divider, Drawer, Input, InputNumber, Select, Space, Switch, Tag } from 'antd';

import {
  getMockState,
  mockScenarioNames,
  resetMockState,
  selectMockScenario,
  setMockFault,
  subscribeMockState,
  updateMockState
} from './state';

function reloadWithoutScenarioOverride() {
  const url = new URL(window.location.href);
  url.searchParams.delete('mockScenario');
  if (url.hash.includes('?')) {
    const [route, query = ''] = url.hash.split('?');
    const params = new URLSearchParams(query);
    params.delete('mockScenario');
    url.hash = params.size ? `${route}?${params}` : route;
  }
  window.location.assign(url.toString());
}

export function MockControlDrawer() {
  const state = useSyncExternalStore(subscribeMockState, getMockState);
  const [open, setOpen] = useState(false);
  const [endpoint, setEndpoint] = useState('/api/vm/info');
  const [status, setStatus] = useState(500);
  const [faultDelay, setFaultDelay] = useState(0);
  const [once, setOnce] = useState(true);

  function switchScenario(scenario: (typeof mockScenarioNames)[number]) {
    selectMockScenario(scenario);
    reloadWithoutScenarioOverride();
  }

  function addFault() {
    const normalized = endpoint.startsWith('/') ? endpoint : `/${endpoint}`;
    setMockFault(normalized, {
      delay: faultDelay,
      enabled: true,
      message: `Injected mock failure for ${normalized}`,
      once,
      status
    });
  }

  return (
    <>
      <Button
        aria-label="Open mock controls"
        className="fixed bottom-5 right-5 z-[1200] shadow-2xl"
        data-testid="mock-controls-button"
        icon={<ApiOutlined />}
        onClick={() => setOpen(true)}
        type="primary"
      >
        Mocks
      </Button>
      <Drawer
        open={open}
        onClose={() => setOpen(false)}
        placement="right"
        title="Mock environment"
        width={380}
      >
        <div className="space-y-5" data-testid="mock-controls">
          <section>
            <div className="mb-2 text-xs font-semibold uppercase tracking-widest text-neutral-500">
              Scenario
            </div>
            <Select
              className="w-full"
              onChange={switchScenario}
              options={mockScenarioNames.map((name) => ({ label: name, value: name }))}
              value={state.scenario}
            />
            <div className="mt-3 flex flex-wrap gap-2">
              <Tag color={state.auth.session.authenticated ? 'green' : 'gold'}>
                {state.auth.session.authenticated ? 'authenticated' : 'signed out'}
              </Tag>
              <Tag>{state.device.hardware}</Tag>
              <Tag color={state.picoclaw.running ? 'blue' : 'default'}>
                PicoClaw {state.picoclaw.running ? 'running' : 'stopped'}
              </Tag>
            </div>
          </section>

          <section>
            <div className="mb-2 text-xs font-semibold uppercase tracking-widest text-neutral-500">
              Global latency
            </div>
            <InputNumber
              addonAfter="ms"
              className="w-full"
              min={0}
              onChange={(value) =>
                updateMockState((current) => {
                  current.delay = value ?? 0;
                })
              }
              step={50}
              value={state.delay}
            />
          </section>

          <Divider />

          <section className="space-y-3">
            <div className="text-xs font-semibold uppercase tracking-widest text-neutral-500">
              Inject API fault
            </div>
            <Input
              aria-label="Fault endpoint"
              onChange={(event) => setEndpoint(event.target.value)}
              placeholder="/api/vm/info"
              value={endpoint}
            />
            <Space.Compact className="w-full">
              <InputNumber
                addonBefore="HTTP"
                className="w-1/2"
                max={599}
                min={400}
                onChange={(value) => setStatus(value ?? 500)}
                value={status}
              />
              <InputNumber
                addonAfter="ms"
                className="w-1/2"
                min={0}
                onChange={(value) => setFaultDelay(value ?? 0)}
                placeholder="delay"
                value={faultDelay}
              />
            </Space.Compact>
            <div className="flex items-center justify-between text-sm">
              <span>Consume after one request</span>
              <Switch checked={once} onChange={setOnce} />
            </div>
            <Button block onClick={addFault} type="primary">
              Inject fault
            </Button>
            {Object.entries(state.faults).map(([path, fault]) => (
              <div
                className="flex items-center justify-between rounded-md border border-neutral-700 px-3 py-2 text-xs"
                key={path}
              >
                <span className="truncate font-mono">{path}</span>
                <Button danger onClick={() => setMockFault(path, null)} size="small" type="text">
                  Clear
                </Button>
                <span className="sr-only">HTTP {fault.status}</span>
              </div>
            ))}
          </section>

          <Divider />

          <Button
            block
            icon={<ReloadOutlined />}
            onClick={() => {
              resetMockState();
              reloadWithoutScenarioOverride();
            }}
          >
            Reset scenario
          </Button>
        </div>
      </Drawer>
    </>
  );
}
