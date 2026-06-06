import { defineConfig } from 'tsup';

// Bundle the SDK (generated client uses extensionless imports designed for a
// bundler) into a single ESM entry + type declarations.
export default defineConfig({
  entry: ['src/index.ts'],
  format: ['esm'],
  dts: true,
  sourcemap: true,
  clean: true,
  treeshake: true,
});
