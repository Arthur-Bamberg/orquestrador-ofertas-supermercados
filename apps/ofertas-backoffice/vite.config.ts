import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      "/api": "http://localhost:8080",
      "/gateway": {
        target: "http://localhost:8090",
        rewrite: (path) => path.replace(/^\/gateway/, ""),
      },
    },
  },
  test: {
    environment: "node",
  },
});
