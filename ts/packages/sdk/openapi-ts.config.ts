import { defineConfig } from '@hey-api/openapi-ts';

// Generates the typed IAM SDK from the OpenAPI 3.1 spec into src/.
// `yarn generate` runs this; `yarn build` then compiles src/ -> dist/.
export default defineConfig({
  input: '../../../openapi/openapi.yaml',
  output: {
    path: 'src',
  },
  plugins: ['@hey-api/client-fetch'],
});
