// Humanizes machine event-type strings for display (e.g. `port_scan` -> `port scan`).
export const formatEventType = (type: string): string => (type ? type.replace(/_/g, ' ') : '')
