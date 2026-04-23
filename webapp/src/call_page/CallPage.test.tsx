// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {render, screen, act} from '@testing-library/react';
import React from 'react';

// Mock RTK SDK before importing CallPage
const mockInitMeeting = jest.fn();
const mockMeeting = {id: 'mock-meeting'};

jest.mock('manifest', () => ({id: 'com.kondo97.mattermost-plugin-rtk'}));

jest.mock('@cloudflare/realtimekit-react', () => ({
    useRealtimeKitClient: jest.fn(() => [null, mockInitMeeting]),
    RealtimeKitProvider: ({children}: {children: React.ReactNode}) => <>{children}</>,
}));

jest.mock('@cloudflare/realtimekit-react-ui', () => ({
    RtkMeeting: () => <div data-testid='call-page-meeting'>{'Meeting'}</div>,
}));

import CallPage from './CallPage';

const mockFetch = jest.fn();

beforeEach(() => {
    jest.clearAllMocks();
    jest.useFakeTimers();

    // Reset useRealtimeKitClient mock to return null meeting by default
    const {useRealtimeKitClient} = require('@cloudflare/realtimekit-react');
    (useRealtimeKitClient as jest.Mock).mockReturnValue([null, mockInitMeeting]);

    // Mock fetch for keepalive leave requests
    mockFetch.mockResolvedValue({ok: true});
    global.fetch = mockFetch;
});

afterEach(() => {
    jest.useRealTimers();
});

describe('CallPage', () => {
    it('renders error screen when token is empty', () => {
        render(
            <CallPage
                token=''
                callId='call1'
            />,
        );
        expect(screen.getByTestId('call-page-error')).toBeTruthy();
        expect(screen.getByTestId('call-page-error').textContent).toBe('Missing call token.');
    });

    it('renders loading state when meeting is null and token provided', () => {
        mockInitMeeting.mockResolvedValue(undefined);
        render(
            <CallPage
                token='valid-token'
                callId='call1'
            />,
        );
        expect(screen.getByTestId('call-page-loading')).toBeTruthy();
    });

    it('renders meeting when SDK initializes', () => {
        const {useRealtimeKitClient} = require('@cloudflare/realtimekit-react');
        (useRealtimeKitClient as jest.Mock).mockReturnValue([mockMeeting, mockInitMeeting]);
        mockInitMeeting.mockResolvedValue(undefined);
        render(
            <CallPage
                token='valid-token'
                callId='call1'
            />,
        );
        expect(screen.getByTestId('call-page-meeting')).toBeTruthy();
    });

    it('calls initMeeting with authToken and audio default', () => {
        mockInitMeeting.mockResolvedValue(undefined);
        render(
            <CallPage
                token='my-token'
                callId='call1'
            />,
        );
        expect(mockInitMeeting).toHaveBeenCalledWith(
            expect.objectContaining({
                authToken: 'my-token',
                defaults: expect.objectContaining({audio: true}),
            }),
        );
    });

    it('does not pass modules to initMeeting (preset manages features)', () => {
        mockInitMeeting.mockResolvedValue(undefined);
        render(
            <CallPage
                token='my-token'
                callId='call1'
            />,
        );
        const callArg = mockInitMeeting.mock.calls[0][0];
        expect(callArg.modules).toBeUndefined();
    });

    it('does NOT call initMeeting when token is empty', () => {
        render(
            <CallPage
                token=''
                callId='call1'
            />,
        );
        expect(mockInitMeeting).not.toHaveBeenCalled();
    });

    it('registers beforeunload handler', () => {
        const addEventSpy = jest.spyOn(window, 'addEventListener');
        mockInitMeeting.mockResolvedValue(undefined);
        render(
            <CallPage
                token='valid-token'
                callId='call1'
            />,
        );
        expect(addEventSpy).toHaveBeenCalledWith('beforeunload', expect.any(Function));
    });

    it('calls fetch with keepalive on beforeunload', () => {
        mockInitMeeting.mockResolvedValue(undefined);
        render(
            <CallPage
                token='valid-token'
                callId='call1'
            />,
        );

        // Trigger beforeunload
        const event = new Event('beforeunload');
        window.dispatchEvent(event);

        expect(mockFetch).toHaveBeenCalledWith(
            '/plugins/com.kondo97.mattermost-plugin-rtk/api/v1/calls/call1/leave',
            expect.objectContaining({
                method: 'POST',
                keepalive: true,
                headers: {'X-Requested-With': 'XMLHttpRequest'},
            }),
        );
    });

    it('shows error screen on SDK initialization failure after retries', async () => {
        mockInitMeeting.mockRejectedValue(new Error('SDK init failed'));
        await act(async () => {
            render(
                <CallPage
                    token='bad-token'
                    callId='call1'
                />,
            );
        });

        // Advance through all 3 retries (2s each)
        // eslint-disable-next-line no-await-in-loop
        for (let i = 0; i < 3; i++) {
            // eslint-disable-next-line no-await-in-loop
            await act(async () => {
                jest.advanceTimersByTime(2000);
            });

            // eslint-disable-next-line no-await-in-loop
            await act(async () => {
                // Flush the rejected promise from the retry
                await Promise.resolve();
            });
        }

        expect(screen.getByTestId('call-page-error')).toBeTruthy();
    });
});
