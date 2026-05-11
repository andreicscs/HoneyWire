export function getCssVariable(variableName) {
    if (typeof window === 'undefined') return '';
    return getComputedStyle(document.documentElement).getPropertyValue(variableName).trim();
}

export function hexToRgb(hex) {
    if (!hex) return '0, 0, 0';
    hex = hex.replace(/^#/, '');
    if (hex.length === 3) {
        hex = hex.split('').map(c => c + c).join('');
    }
    const num = parseInt(hex, 16);
    return `${num >> 16}, ${(num >> 8) & 255}, ${num & 255}`;
}