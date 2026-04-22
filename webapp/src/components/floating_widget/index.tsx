// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {useRealtimeKitClient, RealtimeKitProvider} from '@cloudflare/realtimekit-react';
import {RtkMeeting} from '@cloudflare/realtimekit-react-ui';
import {useLanguage} from '@cloudflare/realtimekit-ui';
import {pluginFetch} from 'client';
import manifest from 'manifest';
import React, {useCallback, useEffect, useRef, useState} from 'react';
import {useIntl} from 'react-intl';
import {useSelector, useDispatch} from 'react-redux';
import {clearMyActiveCall} from 'redux/calls_slice';
import type {FeatureFlags} from 'redux/calls_slice';
import {selectCallByChannel, selectMyActiveCall} from 'redux/selectors';
import jaDict from 'utils/rtk_lang_ja';

const INITIAL_WIDTH = 400;
const INITIAL_HEIGHT = 300;
const INITIAL_RIGHT = 24;
const INITIAL_BOTTOM = 24;

const FloatingWidget = () => {
    const intl = useIntl();
    const dispatch = useDispatch();
    const myActiveCall = useSelector(selectMyActiveCall);
    const activeCall = useSelector(
        myActiveCall ? selectCallByChannel(myActiveCall.channelId) : () => undefined,
    );

    const [meeting, initMeeting] = useRealtimeKitClient();
    const rtkT = useLanguage(intl.locale === 'ja' ? jaDict : undefined);
    const [joinError, setJoinError] = useState<string | null>(null);
    const [isJoining, setIsJoining] = useState(false);
    const [isMinimized, setIsMinimized] = useState(false);
    const [isFullscreen, setIsFullscreen] = useState(false);

    // Drag position state
    const [pos, setPos] = useState({right: INITIAL_RIGHT, bottom: INITIAL_BOTTOM});
    const dragging = useRef(false);
    const dragStart = useRef({mouseX: 0, mouseY: 0, right: INITIAL_RIGHT, bottom: INITIAL_BOTTOM});

    // Retry state for RTK SDK initialization
    const MAX_RETRIES = 3;
    const RETRY_DELAY_MS = 2000;
    const retryCountRef = useRef(0);
    const retryTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

    const attemptInit = useCallback((token: string, flags?: FeatureFlags) => {
        setJoinError(null);
        setIsJoining(true);
        initMeeting({
            authToken: token,
            defaults: {audio: true, video: flags?.video ?? true},
            modules: {
                recording: flags?.recording ?? true,
                chat: flags?.chat ?? true,
                poll: flags?.polls ?? true,
                plugin: flags?.plugins ?? true,
                participant: flags?.participants ?? true,
            },
        }).then((mtg) => {
            // initMeeting resolves after the room is joined. Clear the joining
            // state here so the UI transitions immediately, without relying on
            // the roomJoined event which may fire before our listener is
            // registered (React defers effects until after paint).
            if (flags?.screenShare === false) {
                mtg?.self?.disableScreenShare?.();
            }
            setIsJoining(false);
        }).catch((err: Error) => {
            console.error('[rtk-plugin] Widget RTK init error:', err.message, `(attempt ${retryCountRef.current + 1}/${MAX_RETRIES + 1})`); // eslint-disable-line no-console
            if (retryCountRef.current < MAX_RETRIES) {
                retryCountRef.current += 1;
                console.log(`[rtk-plugin] Retrying initMeeting in ${RETRY_DELAY_MS}ms...`); // eslint-disable-line no-console
                retryTimeoutRef.current = setTimeout(() => attemptInit(token), RETRY_DELAY_MS);
            } else {
                setIsJoining(false);
                setJoinError(err.message);
            }
        });
    }, [initMeeting]); // eslint-disable-line react-hooks/exhaustive-deps

    // Initialize RTK SDK when active call is set
    useEffect(() => {
        if (!myActiveCall?.token) {
            return undefined;
        }
        retryCountRef.current = 0;
        attemptInit(myActiveCall.token, myActiveCall.featureFlags);
        return () => {
            if (retryTimeoutRef.current !== null) {
                clearTimeout(retryTimeoutRef.current);
                retryTimeoutRef.current = null;
            }
        };
    }, [myActiveCall?.callId, myActiveCall?.token, attemptInit]); // eslint-disable-line react-hooks/exhaustive-deps

    // Debug: log meeting state and connection events
    useEffect(() => {
        if (!meeting) {
            return undefined;
        }
        console.log('[rtk-plugin] meeting initialized'); // eslint-disable-line no-console

        // Capture callId at effect time so the handler doesn't close over a stale ref
        const activeCallId = myActiveCall?.callId;

        const onRoomJoined = () => {
            console.log('[rtk-plugin] roomJoined event fired'); // eslint-disable-line no-console
            setJoinError(null);
            setIsJoining(false);
        };
        const onRoomLeft = () => {
            console.log('[rtk-plugin] roomLeft event fired'); // eslint-disable-line no-console
            if (activeCallId) {
                pluginFetch(`/api/v1/calls/${activeCallId}/leave`, {method: 'POST'});
            }
            dispatch(clearMyActiveCall());
        };
        const onMediaConnectionUpdate = (state: unknown) => {
            console.log('[rtk-plugin] mediaConnectionUpdate:', state); // eslint-disable-line no-console
        };

        meeting.self.on('roomJoined', onRoomJoined);
        meeting.self.on('roomLeft', onRoomLeft);
        if ((meeting as any).meta?.on) {
            (meeting as any).meta.on('mediaConnectionUpdate', onMediaConnectionUpdate);
        }

        return () => {
            meeting.self.off('roomJoined', onRoomJoined);
            meeting.self.off('roomLeft', onRoomLeft);
            if ((meeting as any).meta?.off) {
                (meeting as any).meta.off('mediaConnectionUpdate', onMediaConnectionUpdate);
            }
        };
    }, [meeting, myActiveCall?.callId, dispatch]);

    // Exit fullscreen on Escape key
    useEffect(() => {
        if (!isFullscreen) {
            return undefined;
        }
        const onKeyDown = (e: KeyboardEvent) => {
            if (e.key === 'Escape') {
                setIsFullscreen(false);
            }
        };
        window.addEventListener('keydown', onKeyDown);
        return () => window.removeEventListener('keydown', onKeyDown);
    }, [isFullscreen]);

    // Drag handlers
    useEffect(() => {
        const onMouseMove = (e: MouseEvent) => {
            if (!dragging.current) {
                return;
            }
            const dx = e.clientX - dragStart.current.mouseX;
            const dy = e.clientY - dragStart.current.mouseY;
            setPos({
                right: Math.max(0, dragStart.current.right - dx),
                bottom: Math.max(0, dragStart.current.bottom - dy),
            });
        };
        const onMouseUp = () => {
            dragging.current = false;
        };
        window.addEventListener('mousemove', onMouseMove);
        window.addEventListener('mouseup', onMouseUp);
        return () => {
            window.removeEventListener('mousemove', onMouseMove);
            window.removeEventListener('mouseup', onMouseUp);
        };
    }, []);

    // Leave call when browser tab is closed or navigated away (safety net).
    // Uses fetch+keepalive instead of sendBeacon so that auth headers
    // (X-Requested-With) are included — sendBeacon cannot set custom headers
    // and may be rejected by Mattermost CSRF protection.
    useEffect(() => {
        if (!myActiveCall?.callId) {
            return undefined;
        }
        const callId = myActiveCall.callId;
        const handler = () => {
            fetch(`/plugins/${manifest.id}/api/v1/calls/${callId}/leave`, {
                method: 'POST',
                keepalive: true,
                headers: {
                    'X-Requested-With': 'XMLHttpRequest',
                },
            });
        };
        window.addEventListener('beforeunload', handler);
        return () => window.removeEventListener('beforeunload', handler);
    }, [myActiveCall?.callId]);

    if (!myActiveCall || !activeCall) {
        if (myActiveCall) {
            console.warn('[rtk-plugin] FloatingWidget: myActiveCall set but activeCall missing', {callId: myActiveCall.callId, channelId: myActiveCall.channelId, tokenLen: myActiveCall.token?.length}); // eslint-disable-line no-console
        }
        return null;
    }

    const handleMouseDown = (e: React.MouseEvent) => {
        if (isFullscreen) {
            return;
        }
        dragging.current = true;
        dragStart.current = {
            mouseX: e.clientX,
            mouseY: e.clientY,
            right: pos.right,
            bottom: pos.bottom,
        };
    };

    const handleClose = async () => {
        if (meeting) {
            try {
                await meeting.leaveRoom();
            } catch {
                // Ignore leave errors
            }
        }
        await pluginFetch(`/api/v1/calls/${myActiveCall.callId}/leave`, {method: 'POST'});
        dispatch(clearMyActiveCall());
    };

    const containerStyle: React.CSSProperties = isFullscreen ? {
        position: 'fixed',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        zIndex: 1000,
        backgroundColor: '#1e1e2e',
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
    } : {
        position: 'fixed',
        right: pos.right,
        bottom: pos.bottom,
        width: INITIAL_WIDTH,
        zIndex: 1000,
        backgroundColor: '#1e1e2e',
        borderRadius: '8px',
        boxShadow: '0 8px 32px rgba(0,0,0,0.5)',
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
        resize: 'both',
    };

    return (
        <div
            style={containerStyle}
            data-testid='floating-widget'
        >
            {/* Header (drag handle) */}
            <div
                style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: '8px 12px',
                    backgroundColor: '#161622',
                    borderBottom: '1px solid #333',
                    userSelect: 'none',
                    cursor: isFullscreen ? 'default' : 'grab',
                }}
                onMouseDown={handleMouseDown}
            >
                <span style={{color: '#fff', fontSize: '13px', fontWeight: 600}}>
                    {intl.formatMessage({id: 'plugin.rtk.floating_widget.title'}, {id: myActiveCall.callId.slice(0, 8)})}
                </span>
                <div style={{display: 'flex', gap: '4px'}}>
                    {!isFullscreen && (
                        <button
                            type='button'
                            style={headerBtnStyle}
                            title={isMinimized ? intl.formatMessage({id: 'plugin.rtk.floating_widget.expand'}) : intl.formatMessage({id: 'plugin.rtk.floating_widget.minimize'})}
                            onClick={() => setIsMinimized((v) => !v)}
                        >
                            {isMinimized ? '\u25B2' : '\u25BC'}
                        </button>
                    )}
                    <button
                        type='button'
                        style={isFullscreen ? headerBtnActiveStyle : headerBtnStyle}
                        title={isFullscreen ? intl.formatMessage({id: 'plugin.rtk.floating_widget.exit_fullscreen'}) : intl.formatMessage({id: 'plugin.rtk.floating_widget.fullscreen'})}
                        onClick={() => {
                            setIsFullscreen((v) => !v);
                            setIsMinimized(false);
                        }}
                    >
                        {isFullscreen ? '\u2291' : '\u229E'}
                    </button>
                    <button
                        type='button'
                        style={headerBtnStyle}
                        title={intl.formatMessage({id: 'plugin.rtk.floating_widget.leave_call'})}
                        onClick={handleClose}
                        data-testid='floating-widget-leave-call'
                    >
                        {'\u00D7'}
                    </button>
                </div>
            </div>

            {/* Content: RTK Meeting UI */}
            {!isMinimized && (
                <div style={{flex: isFullscreen ? 1 : undefined, height: isFullscreen ? undefined : INITIAL_HEIGHT, overflow: 'hidden'}}>
                    <RealtimeKitProvider value={meeting}>
                        {/* eslint-disable-next-line no-nested-ternary */}
                        {joinError ? (
                            <div style={messageStyle}>
                                <div>
                                    <div>{joinError}</div>
                                    <button
                                        type='button'
                                        onClick={() => {
                                            if (myActiveCall?.token) {
                                                retryCountRef.current = 0;
                                                attemptInit(myActiveCall.token, myActiveCall.featureFlags);
                                            }
                                        }}
                                        style={{
                                            marginTop: '12px',
                                            padding: '6px 16px',
                                            border: '1px solid #aaa',
                                            borderRadius: '4px',
                                            background: 'transparent',
                                            color: '#fff',
                                            cursor: 'pointer',
                                        }}
                                    >
                                        {intl.formatMessage({id: 'plugin.rtk.floating_widget.retry'})}
                                    </button>
                                </div>
                            </div>
                        ) : isJoining || !meeting ? (
                            <div style={messageStyle}>
                                {intl.formatMessage({id: 'plugin.rtk.floating_widget.connecting'})}
                            </div>
                        ) : (
                            <RtkMeeting
                                meeting={meeting}
                                t={rtkT}
                                mode='fill'
                                showSetupScreen={false}
                                style={{width: '100%', height: '100%'}}
                            />
                        )}
                    </RealtimeKitProvider>
                </div>
            )}
        </div>
    );
};

const headerBtnStyle: React.CSSProperties = {
    background: 'none',
    border: 'none',
    color: '#aaa',
    fontSize: '16px',
    cursor: 'pointer',
    lineHeight: 1,
    padding: '0 4px',
};

const headerBtnActiveStyle: React.CSSProperties = {
    ...headerBtnStyle,
    color: '#fff',
    fontSize: '18px',
    background: 'rgba(255,255,255,0.15)',
    borderRadius: '4px',
    padding: '0 6px',
};

const messageStyle: React.CSSProperties = {
    color: '#fff',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    height: '100%',
    padding: '16px',
    textAlign: 'center',
};

export default FloatingWidget;
