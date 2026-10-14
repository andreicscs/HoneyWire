// Shared Tailwind class constants for the FleetManagement view.
//
// Goal: keep the template readable by giving every long utility string a name,
// without changing the rendered output.

export const fleetStyles = {
  pageRoot:
    'min-h-full flex flex-col max-w-[1600px] w-full mx-auto px-2 sm:px-4 lg:px-6 py-4 sm:py-6',
  headerRow: 'flex items-center justify-between shrink-0',
  deployButton: 'gap-2 text-sm transition-all relative',
  // First-startup spotlight applied on top of the deploy button.
  deployAttention:
    'animate-bounce-subtle ring-4 ring-primary-main/30 ring-offset-2 ring-offset-bg shadow-lg',
  emptyState:
    'flex-1 flex items-center justify-center text-center text-base text-text-m py-20',
  nodeGrid: 'grid grid-cols-1 lg:grid-cols-3 2xl:grid-cols-4 gap-5 auto-rows-max',
} as const

// Delay before the loading skeleton appears (avoids flashing on fast loads).
export const FLEET_SKELETON_DELAY_MS = 350
