// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Standalone call page component. No Mattermost framework dependencies.
// No i18n — this bundle runs outside the Mattermost React tree (MAINT-U4-02).

import {useRealtimeKitClient, RealtimeKitProvider} from '@cloudflare/realtimekit-react';
import {RtkMeeting} from '@cloudflare/realtimekit-react-ui';
import manifest from 'manifest';
import React, {useCallback, useEffect, useRef, useState} from 'react';

interface Props {
    token: string;
    callId: string;
    embedded?: boolean;
}

const CallPage = ({token, callId, embedded = false}: Props) => {
    const [meeting, initMeeting] = useRealtimeKitClient();
    const [initError, setInitError] = useState<string | null>(null);

    const MAX_RETRIES = 3;
    const RETRY_DELAY_MS = 2000;
    const retryCountRef = useRef(0);

    const attemptInit = useCallback((authToken: string) => {
        setInitError(null);
        initMeeting({
            authToken,
            defaults: {audio: true, video: true},
        }).catch((err: Error) => {
            // Token intentionally not logged — SEC-U4-01
            console.error('[rtk-plugin] RTK init error:', err.message, `(attempt ${retryCountRef.current + 1}/${MAX_RETRIES + 1})`); // eslint-disable-line no-console
            if (retryCountRef.current < MAX_RETRIES) {
                retryCountRef.current += 1;
                setTimeout(() => attemptInit(authToken), RETRY_DELAY_MS);
            } else {
                setInitError('Failed to connect to the call. Please close this tab and try again.');
            }
        });
    }, [initMeeting, embedded]);

    // Initialize RTK SDK (Pattern U4-5)
    useEffect(() => {
        if (!token) {
            return;
        }
        retryCountRef.current = 0;
        attemptInit(token);
    }, [token, attemptInit]); // eslint-disable-line react-hooks/exhaustive-deps

    // Leave on tab close via sendBeacon (BR-U4-011, US-013)
    // Skip when embedded in iframe — the parent floating widget handles leave.
    useEffect(() => {
        if (!callId || embedded) {
            return undefined;
        }
        const handler = () => {
            navigator.sendBeacon(`/plugins/${manifest.id}/api/v1/calls/${callId}/leave`);
        };
        window.addEventListener('beforeunload', handler);
        return () => window.removeEventListener('beforeunload', handler); // REL-U4-04
    }, [callId, embedded]);

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
        <RealtimeKitProvider
            value={meeting}
            fallback={<div data-testid='call-page-loading'>{'Loading...'}</div>}
        >
            <RtkMeeting
                meeting={meeting}
                mode='fill'
                showSetupScreen={!embedded}
                data-testid='call-page-meeting'
                style={{height: '100vh', width: '100vw'}}
            />
        </RealtimeKitProvider>
    );
};

export default CallPage;
