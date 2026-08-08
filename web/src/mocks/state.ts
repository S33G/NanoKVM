import { createScenario } from './scenarios';
import { mockScenarioNames, type MockFault, type MockScenarioName, type MockState } from './types';

const storageKey = 'nanokvm:mock-state:v1';
const listeners = new Set<() => void>();

function isScenario(value: string | null): value is MockScenarioName {
  return mockScenarioNames.some((name) => name === value);
}

function getScenarioFromUrl() {
  const url = new URL(window.location.href);
  const pageQuery = url.searchParams.get('mockScenario');
  const hashQuery = url.hash.includes('?')
    ? new URLSearchParams(url.hash.slice(url.hash.indexOf('?') + 1)).get('mockScenario')
    : null;
  return isScenario(pageQuery) ? pageQuery : isScenario(hashQuery) ? hashQuery : null;
}

function loadState(): MockState {
  const requestedScenario = getScenarioFromUrl();
  if (requestedScenario) return createScenario(requestedScenario);

  try {
    const saved = sessionStorage.getItem(storageKey);
    if (saved) {
      const parsed = JSON.parse(saved) as MockState;
      if (isScenario(parsed.scenario)) return parsed;
    }
  } catch {
    sessionStorage.removeItem(storageKey);
  }

  return createScenario('pcie-full');
}

let state = loadState();

function persist() {
  sessionStorage.setItem(storageKey, JSON.stringify(state));
  listeners.forEach((listener) => listener());
}

export function getMockState() {
  return state;
}

export function updateMockState(update: (current: MockState) => void) {
  const next = structuredClone(state);
  update(next);
  state = next;
  persist();
}

export function selectMockScenario(scenario: MockScenarioName) {
  state = createScenario(scenario);
  persist();
}

export function resetMockState() {
  state = createScenario(state.scenario);
  persist();
}

export function subscribeMockState(listener: () => void) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function setMockFault(endpoint: string, fault: MockFault | null) {
  updateMockState((current) => {
    if (fault) current.faults[endpoint] = fault;
    else delete current.faults[endpoint];
  });
}

export function consumeMockFault(endpoint: string) {
  const fault = state.faults[endpoint];
  if (!fault?.enabled) return null;
  if (fault.once) {
    state = structuredClone(state);
    delete state.faults[endpoint];
    persist();
  }
  return fault;
}

export { mockScenarioNames } from './types';
export type { MockFault, MockScenarioName, MockState } from './types';
