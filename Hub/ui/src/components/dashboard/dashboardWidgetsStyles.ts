// Shared Tailwind class constants for the dashboard widgets.
//
// Goal: keep the templates readable by giving every long utility string a name,
// without changing the rendered output.

export const filterStyles = {
  bar: 'flex justify-between items-center gap-4',
  scrollOuter: 'flex items-center w-full gap-2',
  viewport: 'flex-1 relative overflow-hidden',
  track: 'relative flex overflow-x-auto gap-1',
  scroll: 'flex overflow-x-auto whitespace-nowrap gap-2 items-center custom-scroll pb-3 pr-2 relative',
  allSticky:
    'sticky left-0 z-20 pr-2 bg-bg flex items-center border-r border-border-default transition-colors duration-normal',
  allPill:
    'shrink-0 px-3.5 py-1.5 rounded-md border text-sm font-medium transition-colors duration-normal shadow-sm outline-none',
  allActive: 'bg-primary-selected text-primary-text border-primary-selected',
  allInactive:
    'bg-secondary-main border-secondary-border text-text-m hover:bg-secondary-hover hover:text-text-h',
  nodePill:
    'shrink-0 relative flex items-center rounded-md border transition-all duration-normal shadow-sm group/pill',
  nodeSelected: 'bg-primary-selected text-primary-text border-primary-selected',
  nodeSelectedSensor:
    'bg-highlight-bg border-highlight-border text-highlight-text ring-1 ring-highlight-ring',
  nodeIdle:
    'bg-secondary-main border-secondary-border text-text-m hover:bg-secondary-hover hover:text-text-h',
  nodeDimmed: 'opacity-70',
  selectZone: 'flex items-center gap-2 pl-2.5 py-1 cursor-pointer flex-1',
  fadeEdge:
    'absolute right-5 top-0 bottom-7 w-10 h-8 bg-gradient-to-l from-bg to-transparent pointer-events-none z-10',
} as const

export const chartWidgetStyles = {
  headRow: 'flex justify-between items-start h-14 relative z-10 shrink-0 w-full',
  title: 'text-base font-medium text-text-h',
  body: 'flex-1 relative mt-2 min-h-0 w-full',
  centerOverlay:
    'absolute inset-0 flex flex-col items-center justify-center pointer-events-none mt-2',
  centerTotal: 'text-3xl transition-colors leading-none',
  centerLabel: 'text-sm text-text-m mt-1 leading-none',
} as const

export const uptimeStyles = {
  scrollArea:
    'absolute top-0 left-0 right-0 bottom-0 overflow-y-auto custom-scroll pr-3 pb-10',
  emptyOverlay:
    'absolute inset-0 flex items-center justify-center text-center text-base text-text-m z-20',
  groupBox: 'transition-all duration-normal rounded-lg p-0.5 mb-0.5 border',
  groupSelected: 'border-select-group-border bg-select-group-bg',
  groupPlain: 'border-transparent',
  groupDimmed: 'opacity-50',
  groupHead: 'px-1.5 mb-1 flex items-center gap-2 group/header',
  sensorRow:
    'flex items-center w-full transition-all duration-normal px-1.5 h-7 rounded-md border',
  sensorSelected: 'bg-select-row-bg border-select-row-border shadow-sm',
  sensorPlain: 'border-transparent',
  sensorDimmed: 'opacity-50',
  blockBase:
    'flex-1 max-w-[8px] min-h-[20px] min-w-[2px] h-5 rounded-[2px] transition-opacity duration-[var(--duration-fast)] hover:opacity-70 cursor-default',
} as const

// Uptime block status -> bar color. Unknown statuses fall back to inset.
export const uptimeBlockColors: Record<string, string> = {
  up: 'bg-success-main',
  down: 'bg-critical',
  degraded: 'bg-high',
  nodata: 'bg-bg-inset',
  pending: 'bg-bg-inset',
}

export const eventTableStyles = {
  card: 'bg-bg-surface border border-border-default rounded-lg overflow-hidden flex flex-col shadow-sm w-full relative z-0',
  headBar:
    'pl-5 pr-4 py-3 border-b border-border-default flex justify-between items-center bg-bg-surface shrink-0',
  exportButton:
    'p-1.5 rounded-md text-text-m transition-colors outline-none hover:bg-secondary-hover hover:text-text-h active:scale-95',
  archiveAllButton:
    'px-2.5 py-1 rounded-md text-base transition-colors shadow-sm border outline-none bg-secondary-main text-secondary-text border-secondary-border hover:bg-archive-bg hover:text-archive-text hover:border-archive-border active:scale-95',
  tableScroll:
    "overflow-x-auto overflow-y-auto custom-scroll max-h-[600px] lg:max-h-[700px] flex-1 after:content-[''] after:block after:h-4 after:shrink-0",
  tableHead:
    'text-base text-text-m sticky top-0 bg-bg-surface z-30 shadow-[0_1px_0_0_var(--color-border-default)] select-none',
  rowBase:
    'hover:bg-secondary-hover cursor-pointer transition-colors duration-[var(--duration-fast)] relative z-0 group',
  rowExpanded: 'bg-bg-base',
  rowBorderOpen: 'border-b border-transparent',
  rowBorderClosed: 'border-b border-border-default',
  archiveRowButton:
    'flex items-center justify-center w-6 h-6 ml-auto rounded-md transition-all duration-[var(--duration-fast)] shadow-sm active:scale-95 border outline-none bg-secondary-main text-secondary-text border-secondary-border hover:bg-archive-bg hover:text-archive-text hover:border-archive-border',
  pageBar:
    'flex items-center justify-between border-t border-border-default bg-bg-surface px-4 py-3 sm:px-5 shrink-0',
  pageButtonMobile:
    'relative inline-flex items-center rounded-md border border-border-default bg-bg-surface px-4 py-2 text-base text-text-h hover:bg-secondary-hover disabled:opacity-50 disabled:cursor-not-allowed transition-colors',
  pageArrow:
    'relative inline-flex items-center px-2 py-1.5 border border-border-default bg-bg-surface text-text-m hover:bg-secondary-hover disabled:opacity-50 disabled:cursor-not-allowed transition-colors focus:z-20 outline-none',
  pageNumber:
    'relative inline-flex items-center px-3.5 py-1.5 text-base border border-border-default transition-colors focus:z-20 outline-none',
  pageActive: 'z-10 bg-primary-selected text-primary-text shadow-inner border-primary-selected',
  pageInactive: 'bg-bg-surface text-text-m hover:bg-secondary-hover hover:text-text-h',
} as const
