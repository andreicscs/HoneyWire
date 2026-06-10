export function getCssVariable(variableName: string): string {
    if (typeof window === 'undefined') return '';
    return getComputedStyle(document.documentElement).getPropertyValue(variableName).trim();
}

export function hexToRgb(hex: string): string {
    if (!hex) return '0, 0, 0';
    hex = hex.replace(/^#/, '');
    if (hex.length === 3) {
        hex = hex.split('').map((c: string) => c + c).join('');
    }
    const num = parseInt(hex, 16);
    return `${num >> 16}, ${(num >> 8) & 255}, ${num & 255}`;
}

export function getComputedRgb(variableName: string): string {
    if (typeof window === 'undefined') return 'rgb(0, 0, 0)';
    
    // 1. Let the browser resolve the CSS variable
    let colorEl = document.getElementById('theme-color-computer');
    if (!colorEl) {
        colorEl = document.createElement('div');
        colorEl.id = 'theme-color-computer';
        colorEl.style.display = 'none';
        document.body.appendChild(colorEl);
    }
    colorEl.style.color = `var(${variableName})`;
    const cssColor = window.getComputedStyle(colorEl).color;

    // 2. Paint a 1x1 pixel on a hidden canvas using the OKLCH color
    const canvas = document.createElement('canvas');
    canvas.width = 1;
    canvas.height = 1;
    const ctx = canvas.getContext('2d', { willReadFrequently: true });
    if (!ctx) return 'rgb(0, 0, 0)';
    
    // The canvas engine natively translates OKLCH to visual pixels
    ctx.fillStyle = cssColor;
    ctx.fillRect(0, 0, 1, 1);

    // 3. Extract the pure RGB integers from the painted pixel!
    const [r, g, b] = ctx.getImageData(0, 0, 1, 1).data;
    
    return `rgb(${r}, ${g}, ${b})`;
}

export function injectAlpha(rgbString: string, alpha: number | string): string {
    if (!rgbString || !rgbString.includes('rgb')) return `rgba(0,0,0,${alpha})`;
    
    // Safely convert rgb(r, g, b) to rgba(r, g, b, alpha)
    if (rgbString.includes('rgba')) {
        return rgbString.replace(/[\d.]+\)$/g, `${alpha})`); 
    }
    return rgbString.replace('rgb', 'rgba').replace(')', `, ${alpha})`);
}