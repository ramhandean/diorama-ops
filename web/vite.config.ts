import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/ws': {
        target: 'ws://127.0.0.1:7437',
        ws: true,
      },
      '/api': 'http://127.0.0.1:7437',
      '/m.js': 'http://127.0.0.1:7437',
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    sourcemap: false,
    chunkSizeWarningLimit: 800,
    rollupOptions: {
      output: {
        manualChunks: {
          'vendor-three': ['three', '@react-three/fiber', '@react-three/drei'],
          'vendor-charts': ['recharts'],
          'vendor-icons': ['lucide-react'],
        },
      },
    },
  },
});
