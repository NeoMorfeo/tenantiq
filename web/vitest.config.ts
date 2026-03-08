import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import path from "path";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  test: {
    environment: "jsdom",
    setupFiles: ["./src/test/setup.ts"],
    globals: true,
    coverage: {
      provider: "v8",
      reporter: ["text", "json"],
      reportsDirectory: "./coverage",
      include: ["src/**"],
      exclude: ["src/test/**", "src/api/**", "src/components/ui/**", "src/i18n/**", "src/routeTree.gen.ts", "src/main.tsx", "src/routes/__root.tsx"],
    },
  },
});
