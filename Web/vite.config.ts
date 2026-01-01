import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig(({ mode }) => {

  const Env = loadEnv(mode, process.cwd(), '')

  return {

    plugins: [react()],

    define: {

      'import.meta.env': Env,
      
    },

  }

})