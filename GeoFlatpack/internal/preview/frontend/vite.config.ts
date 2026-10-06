import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vite';

export default defineConfig({
  base: './',
  worker: { format: 'es' },
  build: {
    outDir: fileURLToPath(new URL('../assets', import.meta.url)),
    emptyOutDir: true,
    assetsInlineLimit: 0,
    chunkSizeWarningLimit: 1100,
  },
});
