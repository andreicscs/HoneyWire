// Shared Tailwind class constants for the Dashboard view.
//
// Goal: keep the template readable by giving every long utility string a name,
// without changing the rendered output.

export const dashboardStyles = {
  page:
    'min-h-full flex flex-col gap-4 sm:gap-6 max-w-[1600px] mx-auto w-full px-2 sm:px-4 lg:px-6 py-4 sm:py-6',
  chartsRow: 'flex flex-wrap gap-4 sm:gap-6 shrink-0',
  chartVelocity:
    'flex-[1_1_350px] min-w-[100%] sm:min-w-[350px] h-[320px] lg:h-[340px] shrink-0',
  chartSeverity:
    'flex-[1_1_280px] min-w-[100%] sm:min-w-[280px] max-w-[450px] mx-auto h-[320px] lg:h-[340px] shrink-0',
  chartUptime:
    'flex-[1.5_1_450px] min-w-[100%] lg:min-w-[450px] h-[320px] lg:h-[340px] shrink-0',
  eventsWrap: 'flex-1 min-h-0 mt-2',
} as const
