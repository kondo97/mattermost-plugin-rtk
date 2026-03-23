// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Standalone call page component. No Mattermost framework dependencies.
// No i18n — this bundle runs outside the Mattermost React tree (MAINT-U4-02).

import React, {useEffect, useState} from 'react';
import {useDyteClient, DyteProvider} from '@cloudflare/realtimekit-react';
import {RtkMeeting} from '@cloudflare/realtimekit-react-ui';

const PLUGIN_ID = 'com.mattermost.plugin-rtk';
const HEARTBEAT_INTERVAL_MS = 15_000;

interface Props {
    token: string;
    callId: string;
}

const CallPage = ({token, callId}: Props) => {
    const [meeting, initMeeting] = useDyteClient();
    const [initError, setInitError] = useState<string | null>(null);

    // Initialize RTK SDK (Pattern U4-5)
    useEffect(() => {
        if (!token) {
            return;
        }
        initMeeting({
            authToken: token,
            defaults: {audio: false, video: false},
        }).catch((err: Error) => {
            // Token intentionally not logged — SEC-U4-01
            console.error('[rtk-plugin] RTK init error:', err.message);
            setInitError('Failed to connect to the call. Please close this tab and try again.');
        });
    }, [token]); // eslint-disable-line react-hooks/exhaustive-deps

    // Heartbeat loop — fire-and-forget every 15s (Pattern U4-3, BR-U4-010, REL-U4-01)
    useEffect(() => {
        if (!callId) {
            return undefined;
        }
        const id = setInterval(() => {
            fetch(`/plugins/${PLUGIN_ID}/api/v1/calls/${callId}/heartbeat`, {method: 'POST'});
        }, HEARTBEAT_INTERVAL_MS);
        return () => clearInterval(id); // REL-U4-03
    }, [callId]);

    // Leave on tab close via sendBeacon (Pattern U4-3, BR-U4-011, US-013)
    useEffect(() => {
        if (!callId) {
            return undefined;
        }
        const handler = () => {
            navigator.sendBeacon(`/plugins/${PLUGIN_ID}/api/v1/calls/${callId}/leave`);
        };
        window.addEventListener('beforeunload', handler);
        return () => window.removeEventListener('beforeunload', handler); // REL-U4-04
    }, [callId]);

    // Missing token — show error screen (BR-U4-007, REL-U4-06)
    if (!token) {
        return (
            <div
                style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    height: '100vh',
                    fontFamily: 'sans-serif',
                    color: '#d24b4e',
                }}
                data-testid='call-page-error'
            >
                {'Missing call token.'}
            </div>
        );
    }

    // SDK initialization error (REL-U4-07)
    if (initError) {
        return (
            <div
                style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    height: '100vh',
                    fontFamily: 'sans-serif',
                    color: '#d24b4e',
                }}
                data-testid='call-page-error'
            >
                {initError}
            </div>
        );
    }

    // Loading state while SDK initializes (USE-U4-01)
    if (!meeting) {
        return (
            <div
                style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    height: '100vh',
                    fontFamily: 'sans-serif',
                }}
                data-testid='call-page-loading'
            >
                {'Connecting...'}
            </div>
        );
    }

    return (
        <DyteProvider
            value={meeting}
            fallback={<div data-testid='call-page-loading'>{'Loading...'}</div>}
        >
            <RtkMeeting
                mode='fill'
                data-testid='call-page-meeting'
                style={{height: '100vh', width: '100vw'}}
            />
        </DyteProvider>
    );
};

export default CallPage;
