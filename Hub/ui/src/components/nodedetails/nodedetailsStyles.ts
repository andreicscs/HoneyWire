// Shared Tailwind class constants for the nodedetails components.
//
// Goal: keep the templates readable by giving every long utility string a name,
// without changing the rendered output.

export const nodeHeaderStyles = {
  root: 'flex flex-col gap-6',
  titleRow: 'flex flex-col sm:flex-row sm:items-start justify-between gap-4',
  aliasTitle:
    'text-[length:var(--text-h1)] font-semibold text-text-h leading-tight truncate max-w-[400px] cursor-edit hover:text-primary-main border-b border-dashed border-transparent hover:border-primary-main transition-colors select-none',
  aliasInput:
    'text-[length:var(--text-h1)] font-semibold text-text-h bg-input-bg border border-primary-main rounded px-2 py-0.5 focus:outline-none ring-1 ring-focus-ring max-w-[400px] truncate',
  metaRow: 'flex flex-wrap items-center gap-x-6 gap-y-2 text-sm text-text-m',
  ipPillBase:
    'flex items-center gap-1.5 transition-colors duration-[var(--duration-fast)] w-max rounded px-1 -ml-1 py-0.5 border border-transparent text-text-m hover:text-text-h hover:bg-secondary-hover',
  ipInput:
    'font-mono text-sm text-text-h bg-input-bg border border-primary-main rounded px-1 py-0 focus:outline-none ring-1 ring-focus-ring w-28',
  divider: 'h-4 w-px bg-border-default hidden sm:block',
  tagPill:
    'px-2 py-0.5 bg-bg-inset/50 border border-border-default text-text-m text-sm rounded-md tracking-wider flex items-center gap-1.5 group/tag transition-colors hover:border-text-m',
  tagInput:
    'pl-5 pr-2 py-0.5 bg-input-bg border border-primary-main text-text-h text-sm rounded-md focus:outline-none ring-1 ring-focus-ring w-28 shadow-sm transition-all placeholder:text-text-m/50',
  addTagButton:
    'px-1.5 py-0.5 border border-dashed border-border-default text-text-m text-sm rounded-md hover:text-text-h hover:border-text-m transition-colors outline-none focus:ring-1 focus:ring-focus-ring',
  actionsRow: 'flex items-center gap-3 shrink-0',
  pendingBanner:
    'flex items-center justify-between w-full max-w-xl bg-high/10 border border-high/30 rounded-lg p-4 transition-all duration-normal',
} as const

export const nodeWidgetsStyles = {
  grid: 'grid grid-cols-1 xl:grid-cols-2 gap-5',
  volumeCard:
    'bg-bg-surface w-full max-w-2xl border border-border-default rounded-lg p-5 shadow-sm flex flex-col',
  volumeList: 'space-y-3 overflow-y-auto custom-scroll max-h-[240px] pr-1',
  emptyText: 'text-sm text-text-m italic',
  activityCard:
    'bg-bg-surface border border-border-default rounded-lg flex flex-col overflow-hidden shadow-sm',
  activityHead:
    'px-4 py-3 border-b border-border-default flex items-center justify-between bg-bg-surface shrink-0',
  viewAllButton: 'text-sm font-medium text-text-m hover:text-text-h transition-colors outline-none',
  tableScroll: 'flex-1 overflow-y-auto custom-scroll bg-bg-surface max-h-[240px]',
  tableHead:
    'text-sm font-medium text-text-m tracking-wider sticky top-0 bg-bg-surface shadow-[0_1px_0_0_var(--color-border-default)]',
  tableCell: 'px-3 py-2 border-b border-border-default',
  activityRow:
    'hover:bg-secondary-hover cursor-default transition-colors duration-[var(--duration-fast)] relative z-0',
} as const

export const nodeGridStyles = {
  title: 'text-sm font-semibold text-text-h mb-4 mt-2',
  grid: 'grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 2xl:grid-cols-5 gap-4',
  card: 'bg-bg-surface border border-border-default rounded-lg p-4 flex flex-col group transition-colors shadow-sm relative overflow-hidden',
  emptyBox:
    'border border-dashed border-border-default rounded-lg p-8 flex flex-col items-center justify-center text-center bg-bg-surface/50',
} as const

export const nodeCatalogStyles = {
  title: 'text-base font-semibold text-text-h mb-4',
  grid: 'grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4',
  stateBox: 'flex flex-col items-center justify-center py-20 text-center',
  card: 'bg-bg-surface border rounded-lg p-4 shadow-sm hover:shadow-md cursor-pointer transition-all duration-normal group flex flex-col relative overflow-hidden',
  cardInstalled: 'border-primary-main/50',
  cardAvailable: 'border-border-default hover:border-primary-main',
} as const

export const nodeModalStyles = {
  overlay:
    'fixed inset-0 z-[var(--z-modal)] flex justify-center items-center p-4 sm:p-6 bg-black/60 backdrop-blur-sm',
  panelBase:
    'bg-bg-base w-full max-h-[85vh] rounded-lg shadow-2xl flex flex-col overflow-hidden border border-border-default transform transition-all',
  closeButton:
    'p-2 -mr-2 text-text-m hover:text-text-h transition-colors duration-[var(--duration-fast)] rounded-full hover:bg-secondary-hover focus:outline-none',
  headRow:
    'px-6 py-5 border-b border-border-default flex justify-between items-center bg-bg-surface shrink-0',
  footerBar:
    'px-6 py-4 border-t border-border-default bg-bg-surface flex justify-end shrink-0',
} as const

export const nodeKeyModalStyles = {
  stack: 'space-y-4',
  keyBox: 'bg-bg-surface border border-border-default rounded-lg p-4 text-sm break-all',
  keyHead: 'flex items-center justify-between mb-2',
  keyValue: 'text-text-m select-all font-mono',
  footer: 'flex justify-end',
} as const

export const sensorModalStyles = {
  headerTop:
    'px-6 py-5 border-b border-border-default flex justify-between items-start bg-bg-surface shrink-0',
  tabBar: 'flex border-b border-border-default px-6 shrink-0 bg-bg-base',
  tabBase:
    'py-3 px-2 text-sm border-b-2 transition-colors duration-[var(--duration-fast)] focus:outline-none',
  tabActive: 'border-primary-main text-text-h font-semibold',
  tabInactive: 'border-transparent text-text-m hover:text-text-h',
  bodyScroll: 'flex-1 overflow-y-auto custom-scroll bg-bg-base',
  severityBox:
    'w-full px-3 py-2 text-sm bg-input-bg border rounded-md cursor-pointer flex justify-between items-center transition-all duration-[var(--duration-fast)] shadow-sm select-none',
  severityBoxOpen: 'border-primary-main ring-1 ring-focus-ring',
  severityBoxClosed: 'border-input-border hover:border-border-default',
  severityOption:
    'px-3 py-2 cursor-pointer transition-colors text-sm font-medium duration-[var(--duration-fast)] flex items-center gap-2',
  severityDropdown:
    'absolute top-full left-0 z-[var(--z-dropdown)] w-full mt-1 bg-bg-surface border border-border-default rounded-md shadow-lg py-1 overflow-hidden',
} as const

// Meatball dropdown item buttons (shared by the header + sensor grid menus).
export const nodeMenuStyles = {
  item: 'w-full text-left px-3 py-2 text-sm font-medium flex items-center gap-2 text-text-m hover:bg-secondary-hover transition-colors group',
  itemDanger:
    'w-full text-left px-3 py-2 text-sm font-medium text-danger-text flex items-center gap-2 hover:bg-danger-bg transition-colors group border-t border-border-default mt-1 pt-2',
  itemUpdate:
    'w-full text-left px-3 py-2 text-sm font-medium text-update-main flex items-center gap-2 hover:bg-update-bg transition-colors group',
} as const
