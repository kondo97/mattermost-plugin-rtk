// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {render, screen, act} from '@testing-library/react';

// Mock RTK SDK before importing CallPage
const mockInitMeeting = jest.fn();
const mockMeeting = {id: 'mock-meeting'};

jest.mock('@cloudflare/realtimekit-react', () => ({
    useDyteClient: jest.fn(() => [null, mockInitMeeting]),
    DyteProvider: ({children}: {children: React.ReactNode}) => <>{children}</>,
}));

jest.mock('@cloudflare/realtimekit-react-ui', () => ({
    RtkMeeting: () => <div data-testid='call-page-meeting'>Meeting</div>,
}));

import CallPage from './CallPage';

const originalSendBeacon = navigator.sendBeacon;

beforeEach(() => {
    jest.clearAllMocks();
    jest.useFakeTimers();
    // Reset useDyteClient mock to return null meeting by default
    const {useDyteClient} = require('@cloudflare/realtimekit-react');
    (useDyteClient as jest.Mock).mockReturnValue([null, mockInitMeeting]);

    // Mock sendBeacon
    Object.defineProperty(navigator, 'sendBeacon', {
        value: jest.fn(),
        writable: true,
        configurable: true,
    });
});

afterEach(() => {
    jest.useRealTimers();
    Object.defineProperty(navigator, 'sendBeacon', {
        value: originalSendBeacon,
        writable: true,
        configurable: true,
    });
});

describe('CallPage', () => {
    it('renders error screen when token is empty', () => {
        render(<CallPage token='' callId='call1' />);
        expect(screen.getByTestId('call-page-error')).toBeTruthy();
        expect(screen.getByTestId('call-page-error').textContent).toBe('Missing call token.');
    });

    it('renders loading state when meeting is null and token provided', () => {
        mockInitMeeting.mockResolvedValue(undefined);
        render(<CallPage token='valid-token' callId='call1' />);
        expect(screen.getByTestId('call-page-loading')).toBeTruthy();
    });

    it('renders meeting when SDK initializes', () => {
        const {useDyteClient} = require('@cloudflare/realtimekit-react');
        (useDyteClient as jest.Mock).mockReturnValue([mockMeeting, mockInitMeeting]);
        mockInitMeeting.mockResolvedValue(undefined);
        render(<CallPage token='valid-token' callId='call1' />);
        expect(screen.getByTestId('call-page-meeting')).toBeTruthy();
    });

    it('calls initMeeting with authToken', () => {
        mockInitMeeting.mockResolvedValue(undefined);
        render(<CallPage token='my-token' callId='call1' />);
        expect(mockInitMeeting).toHaveBeenCalledWith(
            expect.objectContaining({authToken: 'my-token'}),
        );
    });

    it('does NOT call initMeeting when token is empty', () => {
        render(<CallPage token='' callId='call1' />);
        expect(mockInitMeeting).not.toHaveBeenCalled();
    });

    it('registers beforeunload handler for sendBeacon', () => {
        const addEventSpy = jest.spyOn(window, 'addEventListener');
        mockInitMeeting.mockResolvedValue(undefined);
        render(<CallPage token='valid-token' callId='call1' />);
        expect(addEventSpy).toHaveBeenCalledWith('beforeunload', expect.any(Function));
    });

    it('calls sendBeacon with correct URL on beforeunload', () => {
        mockInitMeeting.mockResolvedValue(undefined);
        render(<CallPage token='valid-token' callId='call1' />);

        // Trigger beforeunload
        const event = new Event('beforeunload');
        window.dispatchEvent(event);

        expect(navigator.sendBeacon).toHaveBeenCalledWith(
            '/plugins/com.mattermost.plugin-rtk/api/v1/calls/call1/leave',
        );
    });

    it('shows error screen on SDK initialization failure', async () => {
        mockInitMeeting.mockRejectedValue(new Error('SDK init failed'));
        await act(async () => {
            render(<CallPage token='bad-token' callId='call1' />);
        });
        expect(screen.getByTestId('call-page-error')).toBeTruthy();
    });
});
