// Shared Tailwind class constants for the NodeDetails view.
//
// Goal: keep the template readable by giving every long utility string a name,
// without changing the rendered output.

export const nodeDetailsStyles = {
  pageRoot:
    'min-h-full flex flex-col max-w-[1600px] w-full mx-auto px-2 sm:px-4 lg:px-6 py-4 sm:py-6',
  backRow: 'mt-4 sm:mt-6 mb-4 shrink-0',
  backButton:
    'flex items-center gap-1.5 text-sm font-medium text-text-m hover:text-text-h transition-colors outline-none w-max',
  nodeCard:
    'bg-bg-base border border-border-default rounded-lg p-5 sm:p-6 mb-8 shrink-0 shadow-sm flex flex-col gap-6',
} as const
