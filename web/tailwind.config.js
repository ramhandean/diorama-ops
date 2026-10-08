/** @type {import('tailwindcss').Config} */
export default {
  darkMode: ['class'],
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        background: '#09090b',
        surface: {
          DEFAULT: '#0c0c0f',
          subtle: '#121216',
          muted: '#18181c',
        },
        border: {
          DEFAULT: '#27272a',
          subtle: '#18181b',
        },
        foreground: {
          DEFAULT: '#fafafa',
          muted: '#a1a1aa',
          dim: '#71717a',
        },
        accent: {
          DEFAULT: '#3b82f6',
          hover: '#2563eb',
        },
      },
      borderRadius: {
        lg: '0.5rem',
        md: '0.375rem',
        sm: '0.25rem',
      },
    },
  },
  plugins: [],
};
