/**
 * Tarhiata Cloud Studio — Reactive State Store
 * Leaf module: zero internal dependencies.
 */

export const state = {
    servers: [],
    activeServer: null,
    selectedServerName: null,
    selectedInspection: null,
    currentHostServices: [],
    swarmServicesCache: [],
    swarmDatabasesCache: [],
    swarmNodesCache: [],
    currentServiceLinks: [],
    modalMode: 'local',
    dbDeployMode: 'single-node',
    currentActiveTab: 'tabSwarmServices',
    isTestingAll: false,
    isPollingTelemetry: false
};
