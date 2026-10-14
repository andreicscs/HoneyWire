// Shared Tailwind class constants for the fleetmanagement components.
//
// Goal: keep the templates readable by giving every long utility string a name,
// without changing the rendered output. Menu + IP/tag pill patterns are shared
// with the nodedetails components (see nodedetailsStyles.ts).

export const fleetWidgetStyles = {
  frame:
    'flex flex-col h-full min-h-[280px] transition-all duration-[var(--duration-fast)] !overflow-visible hover:z-elevated focus-within:z-elevated',
  headerRow: 'flex items-start justify-between w-full',
  aliasTitle:
    'text-base font-medium text-text-h truncate max-w-[180px] cursor-edit hover:text-primary-main border-b border-dashed border-transparent hover:border-primary-main transition-colors select-none',
  aliasInput:
    'text-base font-medium text-text-h bg-input-bg border border-primary-main rounded px-1.5 py-0 focus:outline-none ring-1 ring-focus-ring max-w-[180px] truncate',
  bodyWrap: 'flex-1 mt-3 flex flex-col',
  ipGrid: 'grid grid-cols-2 gap-y-2 gap-x-4 text-sm mb-4',
  tagInput:
    'pl-5 pr-2 py-0.5 bg-input-bg border border-primary-main text-text-h text-sm rounded-md focus:outline-none ring-1 ring-focus-ring w-32 shadow-sm transition-all placeholder:text-text-m/50',
  tagRow: 'flex flex-wrap gap-1.5 mb-4',
  sensorBox: 'mt-auto bg-bg-surface border border-border-default rounded-lg p-3',
  summaryPill:
    'px-2 py-1 rounded-md text-sm font-medium flex items-center gap-1.5 border border-border-default bg-secondary-main text-text-m cursor-default hover:border-text-m transition-colors',
  tooltipBox:
    'absolute bottom-full left-0 mb-2 w-max min-w-[180px] opacity-0 invisible group-hover/tooltip:opacity-100 group-hover/tooltip:visible transition-all duration-fast z-dropdown bg-bg-surface border border-border-default shadow-lg rounded-md p-2 pointer-events-none',
  skeletonPill:
    'px-2 py-1 rounded-md text-sm bg-secondary-main border border-border-default animate-pulse w-16 h-6',
  awaitingBox:
    'mt-auto bg-disabled-bg border border-dashed border-disabled-border rounded-lg p-4 flex flex-col items-center justify-center text-center opacity-70',
  footerButton:
    'w-full py-3 text-sm font-medium text-text-h hover:text-text-h bg-bg-secondary hover:bg-secondary-hover border-t border-border-default rounded-b-xl transition-colors flex items-center justify-center gap-2 group/btn outline-none',
} as const

export const fleetDeployStyles = {
  formStack: 'space-y-5',
  intro: 'text-sm text-text-m leading-normal',
  fieldLabel: 'block text-sm font-medium text-text-h mb-1.5',
  formFooter: 'flex justify-end gap-3 pt-5 border-t border-border-default mt-6',
  successStack: 'space-y-6',
  successBanner:
    'flex items-center gap-3 text-success-text bg-success-bg border border-success-border p-3.5 rounded-md',
  cmdHeadRow: 'flex items-center justify-between mb-1.5',
  versionBadge:
    'px-1.5 py-0.5 text-[10px] font-mono font-medium rounded bg-secondary-main text-text-m border border-secondary-border',
  cmdCode:
    'block w-full p-4 bg-bg-inset/50 border border-border-default rounded-md text-sm font-mono text-text-h whitespace-pre-wrap break-all leading-normal select-all',
  keyCode:
    'flex-1 block px-3 py-2.5 bg-bg-inset/50 border border-border-default rounded-md text-sm font-mono text-text-m truncate select-all',
  keyHint: 'text-sm text-success-main font-medium mt-2',
  doneFooter: 'flex justify-end pt-5 border-t border-border-default mt-6',
} as const

export const fleetSkeletonStyles = {
  grid: 'grid grid-cols-1 lg:grid-cols-3 2xl:grid-cols-4 gap-5 auto-rows-max',
  card: 'bg-bg-surface border border-border-default rounded-lg overflow-hidden shadow-sm flex flex-col min-h-[280px]',
  pulse: 'bg-bg-inset rounded animate-pulse',
} as const
