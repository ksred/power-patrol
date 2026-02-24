/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // Base - Soft charcoal grays
        'bg-primary': '#0f0f0f',
        'bg-secondary': '#171717',
        'bg-elevated': '#1f1f1f',
        'bg-card': '#1a1a1a',
        'bg-hover': '#262626',

        // Semantic - Muted, harmonious
        'cpu': '#f97316',      // Soft orange
        'gpu': '#a78bfa',      // Soft violet
        'ane': '#22d3ee',      // Soft cyan
        'battery': '#34d399',  // Soft emerald
        'warning': '#fbbf24',  // Amber
        'danger': '#f87171',   // Soft red
        'success': '#4ade80',  // Green
      },
      fontFamily: {
        sans: ['Geist', '-apple-system', 'BlinkMacSystemFont', 'sans-serif'],
        mono: ['Geist Mono', 'SF Mono', 'monospace'],
      },
      borderColor: {
        subtle: 'rgba(255, 255, 255, 0.06)',
        default: 'rgba(255, 255, 255, 0.1)',
        focus: 'rgba(255, 255, 255, 0.15)',
      },
      animation: {
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        'fade-in': 'fadeIn 0.2s ease-out',
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0', transform: 'translateY(8px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
      },
    },
  },
  plugins: [],
}
