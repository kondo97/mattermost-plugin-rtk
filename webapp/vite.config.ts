import path from 'path';

import {defineConfig} from 'vite';
import react from '@vitejs/plugin-react';

// Mattermost host provides these as browser globals.
// Applied to the 'main' entry only — the 'call' entry is a standalone bundle
// and must include React independently.
const MATTERMOST_EXTERNALS: Record<string, string> = {
    react: 'React',
    'react-dom': 'ReactDOM',
    redux: 'Redux',
    'react-redux': 'ReactRedux',
    'prop-types': 'PropTypes',
    'react-bootstrap': 'ReactBootstrap',
    'react-router-dom': 'ReactRouterDom',
};

// Set VITE_BUILD_TARGET=call to build the standalone call page bundle.
// Default (no env var) builds the Mattermost plugin main bundle.
const buildTarget = process.env.VITE_BUILD_TARGET ?? 'main';
const isCallBuild = buildTarget === 'call';

export default defineConfig({
    plugins: [react()],

    resolve: {
        alias: {
            // Allow imports relative to src/ root (matches webpack resolve.modules config)
            src: path.resolve(__dirname, 'src'),
        },
    },

    define: {
        // Ensure process.env.NODE_ENV is defined for third-party packages
        'process.env.NODE_ENV': JSON.stringify(process.env.NODE_ENV ?? 'production'),
    },

    build: {
        outDir: 'dist',

        // Only clean dist/ on the main build so call.js is not deleted
        emptyOutDir: !isCallBuild,

        rollupOptions: {
            input: isCallBuild
                ? {call: path.resolve(__dirname, 'src/call_page/main.tsx')}
                : {main: path.resolve(__dirname, 'src/index.tsx')},

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
