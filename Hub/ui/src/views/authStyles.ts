// Shared Tailwind class constants for the auth views (Login + Setup).
//
// Goal: keep the templates readable by giving every long utility string a name,
// without changing the rendered output.

export const authStyles = {
  pageFixed:
    'h-screen flex flex-col items-center justify-center bg-bg-base p-6 relative transition-colors duration-normal',
  pageScroll:
    'min-h-screen flex flex-col items-center justify-center bg-bg-base p-6 relative py-12 overflow-y-auto transition-colors duration-normal',
  themeToggle: 'absolute top-6 right-6',
  containerSm: 'w-full max-w-sm z-10',
  containerXl: 'w-full max-w-xl z-10 my-auto',
  headerBlock: 'mb-10',
  card: 'relative overflow-hidden transition-all duration-normal',
  severityBar: 'absolute top-0 left-0 right-0 h-1 opacity-90',
  formTight: 'mt-2 space-y-6',
  formRelaxed: 'mt-4 space-y-6',
  setupGrid: 'grid grid-cols-1 sm:grid-cols-2 gap-4',
  setupStack: 'space-y-6',
  setupSubmitWrap: 'pt-4',
} as const

export const SEVERITY_GRADIENT =
  'linear-gradient(to right, var(--sev-info), var(--sev-low), var(--sev-medium), var(--sev-high), var(--sev-critical))'
