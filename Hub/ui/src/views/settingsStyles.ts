// Shared Tailwind class constants for the Settings view.
//
// Goal: keep the template readable by giving every long utility string a name,
// without changing the rendered output. Values below are verbatim copies of the
// classes that were previously inline (whitespace normalised).

export const settingsLayout = {
  pageRoot:
    'h-full flex flex-col max-w-[1600px] w-full mx-auto px-2 sm:px-4 lg:px-6 pt-4 sm:pt-6 transition-colors duration-200',
  contentRow: 'flex flex-col md:flex-row gap-6 flex-1 min-h-0',
  scrollRegion:
    "flex-1 overflow-y-auto custom-scroll pr-2 space-y-6 after:content-[''] after:block after:h-24 after:shrink-0",
} as const

export const settingsHeader = {
  saveError: 'text-xs text-danger-main hidden sm:block',
  saveSuccess: 'text-xs text-success-main hidden sm:block',
  // NOTE: `text-medium` / `bg-medium` are not design-system tokens (probably
  // meant to be `text-text-m`). Preserved verbatim to avoid visual changes.
  unsavedHint: 'text-xs text-medium animate-pulse hidden sm:flex items-center gap-1.5',
  unsavedDot: 'w-2 h-2 rounded-full bg-medium inline-block',
} as const

export const settingsForm = {
  narrowStack: 'max-w-md space-y-4',
  narrow: 'max-w-md',
  wide: 'max-w-xl',
  sectionStack: 'space-y-6',
  fieldStack: 'space-y-4',
  sectionTitle: 'text-sm text-text-m font-medium mb-4',
} as const

export const settingsWarning = {
  box: 'flex items-start gap-2 text-danger-text bg-danger-bg p-3 rounded-md border border-danger-border mt-4',
  icon: 'w-5 h-5 shrink-0 mt-0.5',
  body: 'text-sm',
  title: 'font-medium',
  desc: 'mt-0.5 opacity-90',
} as const

export type UpdateFeedbackType = 'success' | 'info' | 'warning' | 'error'

export const settingsUpdateCheck = {
  divider: 'pt-3 border-t border-border-default/60 flex flex-col gap-2.5',
  row: 'flex items-center justify-between gap-4 max-w-md',
  textWrap: 'flex-1 min-w-0 pr-2',
  label: 'block text-xs text-text-h mb-1',
  desc: 'text-xs text-text-m leading-normal',
  button:
    'w-[112px] justify-center text-sm tracking-wider py-1.5 rounded-md transition-all border outline-none select-none bg-secondary-main text-secondary-text border-secondary-border hover:bg-secondary-hover hover:text-text-h shadow-sm cursor-pointer flex items-center gap-2 group disabled:opacity-50 disabled:cursor-not-allowed shrink-0',
  spinnerIcon: 'w-3.5 h-3.5 animate-spin',
  refreshIcon: 'w-3.5 h-3.5 transition-transform duration-normal group-hover:rotate-180',
  feedbackBase:
    'w-fit inline-flex items-center gap-2 py-1.5 px-3 rounded-md text-xs border transition-all mt-0.5',
  feedbackVariant: {
    success: 'bg-success-bg text-success-text border-success-border',
    info: 'bg-update-bg text-update-main border-update-border',
    warning: 'bg-warning-bg text-warning-text border-warning-border',
    error: 'bg-danger-bg text-danger-text border-danger-border',
  } satisfies Record<UpdateFeedbackType, string>,
  feedbackIcon: {
    success: 'w-3.5 h-3.5 shrink-0 text-success-main',
    info: 'w-3.5 h-3.5 shrink-0 text-update-main',
    warning: 'w-3.5 h-3.5 shrink-0 text-warning-main',
    error: 'w-3.5 h-3.5 shrink-0 text-danger-main',
  } satisfies Record<UpdateFeedbackType, string>,
  feedbackText: 'text-xs leading-normal',
} as const

export function updateFeedbackVariantClass(type: string): string {
  return (
    settingsUpdateCheck.feedbackVariant[type as UpdateFeedbackType] ??
    settingsUpdateCheck.feedbackVariant.error
  )
}

export function updateFeedbackIconClass(type: string): string {
  return (
    settingsUpdateCheck.feedbackIcon[type as UpdateFeedbackType] ??
    settingsUpdateCheck.feedbackIcon.error
  )
}

export const settingsSeverity = {
  label: 'block text-sm tracking-wider text-text-m mb-2',
  row: 'flex flex-wrap gap-2.5',
  button:
    'text-sm tracking-wider px-3.5 py-1.5 rounded-md transition-all border outline-none select-none hover:opacity-80',
} as const

export const settingsSecurity = {
  panel: 'space-y-6',
  desc: 'text-sm text-text-m mb-4 max-w-2xl',
  dangerTitle: 'text-base text-danger-text -mt-10 mb-4 ml-7',
  dangerRow:
    'flex flex-col sm:flex-row sm:items-center justify-between gap-4 bg-bg-surface border border-danger-border p-5 rounded-lg shadow-sm transition-colors',
  dangerName: 'text-base text-text-h',
  dangerDesc: 'text-xs text-text-m mt-1 max-w-xl',
  shrink: 'shrink-0',
  icon: 'w-5 h-5 shrink-0',
} as const

export const settingsModal = {
  form: 'space-y-4',
  error: 'text-xs text-danger-text bg-danger-bg p-2.5 rounded-md border border-danger-border',
  errorCentered:
    'text-xs text-danger-text bg-danger-bg p-2.5 rounded-md border border-danger-border text-center',
  footer: 'pt-4 flex justify-end gap-3',
  confirmCopy: 'text-sm text-text-m mb-5',
  icon: 'w-5 h-5 shrink-0',
  iconLg: 'w-6 h-6 shrink-0',
} as const

// Dimmed state for disabled retention steppers.
export const disabledInputClass = '[&_input]:!text-disabled-text'
