import { defineConfig } from 'vite';
import { DASHBOARD_SECURITY_HEADERS } from './src/cspPolicy';

export default defineConfig({
  server: {
    headers: DASHBOARD_SECURITY_HEADERS,
  },
  preview: {
    headers: DASHBOARD_SECURITY_HEADERS,
  },
});
