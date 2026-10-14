// Shared Tailwind class constants for the layout components (Sidebar + Header).
//
// Goal: keep the templates readable by giving every long utility string a name,
// without changing the rendered output.

export const sidebarStyles = {
  aside:
    'flex flex-col bg-bg-base border-r border-border-default --z-modal transition-all duration-fast ease-in-out shrink-0',
  asideOpen: 'w-[240px]',
  asideClosed: 'w-[68px]',
  brandRow: 'h-14 flex items-center px-[18px] shrink-0 mb-2',
  collapseButton:
    'p-1.5 rounded-md text-secondary-text text-text-h hover:bg-secondary-hover transition-colors outline-none flex items-center justify-center',
  nav: 'flex-1 space-y-2 overflow-y-auto custom-scroll overflow-x-hidden px-3',
  sectionLabel:
    'px-3 text-xs text-text-m transition-all duration-fast overflow-hidden whitespace-nowrap',
  sectionOpen: 'max-h-6 opacity-100 mb-1',
  sectionClosed: 'max-h-0 opacity-0 mb-0',
  navButton:
    'w-full flex items-center py-2.5 px-3 justify-start rounded-md text-base text-text-h transition-all border outline-none',
  navActive: 'bg-secondary-selected shadow-sm border-secondary-border',
  navInactive:
    'border-transparent text-secondary-text hover:bg-secondary-hover hover:text-text-h',
  navLabel:
    'overflow-hidden transition-all duration-fast ease-in-out whitespace-nowrap flex items-center',
  navLabelOpen: 'max-w-[150px] ml-3 opacity-100',
  navLabelClosed: 'max-w-0 ml-0 opacity-0',
  versionLink:
    'flex justify-center items-center text-[10px] pb-2 transition-all duration-fast ease-in-out group',
  versionUpdate: 'text-update-main hover:text-update-hover cursor-pointer',
  versionPlain: 'text-text-m/50',
  footer: 'relative p-3 shrink-0 space-y-2',
  archiveButton:
    'w-full flex items-center justify-center py-2 px-3 rounded-md text-xs transition-all border shadow-sm outline-none',
  archiveOn: 'bg-archive-bg text-archive-text border-archive-border hover:bg-archive-hover',
  archiveOff:
    'bg-secondary-main text-secondary-text border-secondary-border hover:bg-archive-bg hover:text-archive-text',
  footerLabel:
    'overflow-hidden transition-all duration-fast ease-in-out whitespace-nowrap flex items-center',
  footerLabelOpen: 'max-w-[120px] ml-2 opacity-100',
  footerLabelClosed: 'max-w-0 ml-0 opacity-0',
  purgeButton:
    'w-full flex items-center justify-center py-2 px-3 rounded-md text-xs transition-all shadow-sm border outline-none bg-danger-bg text-danger-text border-danger-border hover:bg-danger-hover',
} as const

export const headerStyles = {
  bar: 'h-14 bg-gradient-to- from-bg to-transparent flex items-center justify-between px-4 sm:px-6 shrink-0 backdrop-blur-sm',
  brandGroup: 'flex items-center gap-4',
  logoGroup: 'flex items-center gap-2 pr-4 mr-2 border-r border-border-default',
  routeTitle: 'text-base text-text-m font-medium capitalize hidden sm:block',
  actions: 'flex items-center gap-3',
  unreadButton:
    'hidden sm:flex items-center gap-2 px-2.5 py-1 rounded-md text-xs mr-1 bg-danger-bg text-danger-text border border-danger-border transition-colors hover:bg-danger-hover focus:outline-none shadow-sm',
  armedBase: 'px-3 py-1.5 rounded-md text-xs transition-colors border shadow-sm focus:outline-none',
  armedOn: 'bg-success-bg text-success-text border-success-border hover:bg-success-selected',
  armedOff:
    'bg-secondary-main text-secondary-text border-secondary-border hover:bg-secondary-hover hover:text-text-h',
  iconButton:
    'w-8 h-8 rounded-md bg-secondary-main border border-secondary-border text-secondary-text hover:text-text-h hover:bg-secondary-hover transition-colors flex items-center justify-center group shadow-sm focus:outline-none',
  divider: 'w-px h-4 bg-border-default mx-1',
  exitButton:
    'text-xs text-secondary-text hover:text-text-h transition-colors cursor-pointer bg-transparent border-0 p-0 focus:outline-none',
  modalBody: 'p-6 md:p-8 space-y-4',
  updateBadge:
    'px-2.5 py-1 bg-update-bg border border-update-main/30 text-update-text text-xs font-mono font-medium rounded-md flex items-center gap-2 group/tag',
  cmdBox:
    'bg-bg-inset/50 border border-border-default rounded-md p-4 relative group flex flex-col gap-3',
} as const
