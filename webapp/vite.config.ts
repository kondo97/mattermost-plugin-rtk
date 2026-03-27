import path from 'path';

import react from '@vitejs/plugin-react';
import cssInjectedByJs from 'vite-plugin-css-injected-by-js';
import {defineConfig} from 'vite';
import type {Plugin} from 'vite';

// Mattermost's CSP blocks blob: URLs for Web Workers.
// worker-timers (a dependency of @cloudflare/realtimekit) creates a worker from a blob: URL.
// This plugin patches the worker creation to use a static URL served by the Go plugin instead,
// which satisfies the CSP's 'self' directive.
function workerTimersCspPatch(): Plugin {
    const workerUrl = '/plugins/com.kondo97.mattermost-plugin-rtk/worker.js';
    return {
        name: 'worker-timers-csp-patch',
        transform(code, id) {
            if (!id.includes('worker-timers') || !id.includes('load-or-return-broker')) {
                return null;
            }
            const patched = code.replace(
                /const blob[\s\S]*?URL\.revokeObjectURL\(url\)\);/,
                `broker = loadBroker('${workerUrl}');`,
            );
            if (patched === code) {
                console.warn('[worker-timers-csp-patch] Pattern not found - check worker-timers version'); // eslint-disable-line no-console
            }
            return patched;
        },
    };
}

// Mattermost host provides these as browser globals.
// Applied to the 'main' entry only — the 'call' entry is a standalone bundle
// and must include React independently.
const MATTERMOST_EXTERNALS: Record<string, string> = {
    react: 'React',
    'react-dom': 'ReactDOM',
    redux: 'Redux',
    'react-redux': 'ReactRedux',
    'react-intl': 'ReactIntl',
    'prop-types': 'PropTypes',
    'react-bootstrap': 'ReactBootstrap',
    'react-router-dom': 'ReactRouterDom',
};

// Set VITE_BUILD_TARGET=call to build the standalone call page bundle.
// Default (no env var) builds the Mattermost plugin main bundle.
const buildTarget = process.env.VITE_BUILD_TARGET ?? 'main'; // eslint-disable-line no-process-env
const isCallBuild = buildTarget === 'call';

export default defineConfig({
    plugins: [react(), cssInjectedByJs(), workerTimersCspPatch()],

    resolve: {
        alias: [

            // Resolve src-rooted bare imports, replicating webpack's resolve.modules: ['src'].
            // e.g. 'redux/calls_slice' → src/redux/calls_slice.ts
            //      'components/switch_call_modal' → src/components/switch_call_modal/index.tsx
            {
                find: /^(redux|components|utils|call_page)\/(.+)$/,
                replacement: `${path.resolve(__dirname, 'src')}/$1/$2`,
            },
            {find: 'client', replacement: path.resolve(__dirname, 'src/client')},
            {find: 'manifest', replacement: path.resolve(__dirname, 'src/manifest')},

            // Keep the 'src' alias for explicit src/... imports
            {find: 'src', replacement: path.resolve(__dirname, 'src')},
        ],
    },

    define: {

        // Ensure process.env.NODE_ENV is defined for third-party packages
        'process.env.NODE_ENV': JSON.stringify(process.env.NODE_ENV ?? 'production'), // eslint-disable-line no-process-env
    },

    build: {
        outDir: 'dist',

        // Only clean dist/ on the main build so call.js is not deleted
        emptyOutDir: !isCallBuild,

        rollupOptions: {
            input: isCallBuild ?
                {call: path.resolve(__dirname, 'src/call_page/main.tsx')} :
                {main: path.resolve(__dirname, 'src/index.tsx')},

            output: {
                entryFileNames: '[name].js',
                chunkFileNames: 'chunk-[hash].js',
                format: 'iife',

                // globals maps external module IDs to their browser global names.
                // Only applies to the main entry (call entry has no externals).
                ...(isCallBuild ? {} : {globals: MATTERMOST_EXTERNALS}),
            },

            // Externalize Mattermost-provided globals for the main entry only.
            // The call entry bundles everything including React.
            external: isCallBuild ? [] : Object.keys(MATTERMOST_EXTERNALS),
        },
    },
});
