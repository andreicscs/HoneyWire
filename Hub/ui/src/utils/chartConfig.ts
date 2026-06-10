// src/utils/chartConfig.js
import { getComputedRgb, injectAlpha } from './theme'

export const baseTooltipConfig: Record<string, any> = {
    borderWidth: 1, 
    padding: 10, 
    boxPadding: 6, 
    usePointStyle: true, 
    boxWidth: 8, 
    boxHeight: 8, 
    titleFont: { size: 11, family: 'ui-monospace, monospace', weight: 'normal' }, 
    bodyFont: { size: 12, family: 'Inter, sans-serif', weight: 'bold' }
}

export const applyChartTheme = (chartInstance: any): void => {
    if (!chartInstance) return;
    
    // Dynamically pull colors from the DOM
    const bgSurfaceRgb = getComputedRgb('--bg-surface');
    
    // Apply styling to the native tooltip
    chartInstance.options.plugins.tooltip.backgroundColor = injectAlpha(bgSurfaceRgb, 0.95);
    chartInstance.options.plugins.tooltip.titleColor = getComputedRgb('--text-m');
    chartInstance.options.plugins.tooltip.bodyColor = getComputedRgb('--text-h');
    chartInstance.options.plugins.tooltip.borderColor = getComputedRgb('--border-default');
    
    // Update X-axis ticks if the chart has them
    if (chartInstance.options.scales?.x) {
        chartInstance.options.scales.x.ticks.color = getComputedRgb('--text-m');
    }

    chartInstance.update('none');
}