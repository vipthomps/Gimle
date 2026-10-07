import { defineConfig } from 'vite'
import solid from 'vite-plugin-solid'

export default defineConfig({
  plugins: [solid()],
  build: {
    rollupOptions: {
      output: {
        // index.js and index.css keep fixed names because the page links
        // them; the server tells browsers to check them on every load. Page
        // chunks carry a hash so a new version never mixes with a cached one.
        entryFileNames: `assets/[name].js`,
        chunkFileNames: `assets/[name]-[hash].js`,
        assetFileNames: info => info.names?.[0] == 'index.css' ? `assets/[name].[ext]` : `assets/[name]-[hash].[ext]`
      }
    }
  }
})
