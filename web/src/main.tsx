import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import './index.css';
import '@fontsource/ibm-plex-mono/400.css';
import '@/lib/sdk'; // configure the SDK client (side-effect)
import { App } from './App';

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
