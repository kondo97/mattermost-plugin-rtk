// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {shallow, mount} from 'enzyme';
import React from 'react';
import {act} from 'react-dom/test-utils';
import {useSelector, useDispatch} from 'react-redux';

import ChannelHeaderButton from './index';

// Mock react-redux hooks
jest.mock('react-redux', () => ({
    useSelector: jest.fn(),
    useDispatch: jest.fn(),
}));

// Mock pluginFetch
jest.mock('client', () => ({
    pluginFetch: jest.fn(),
}));

// Mock manifest
jest.mock('manifest', () => ({id: 'com.mattermost.plugin-rtk'}));

// Mock react-intl
jest.mock('react-intl', () => ({
    useIntl: () => ({
        formatMessage: ({id}: {id: string}) => id,
    }),
}));

// Mock SwitchCallModal
jest.mock('components/switch_call_modal', () => () => null);

const mockDispatch = jest.fn();
const channel = {id: 'channel1'} as never;
const currentUserId = 'currentUser';

const setSelectors = ({
    pluginEnabled = true,
    activeCall = undefined as object | undefined,
    myActiveCall = null as object | null,
    isParticipant = false,
    channelDisplayName = 'general',
} = {}) => {
    (useSelector as unknown as jest.Mock).mockImplementation(() => {
        // We identify selectors by call order within each render cycle (5 selectors per render).
        // Use modulo so re-renders (calls 5-9, 10-14, …) return the same values.
        const callCount = (useSelector as unknown as jest.Mock).mock.calls.length;
        const idx = (callCount - 1) % 5;
        if (idx === 0) {
            return pluginEnabled;
        }
        if (idx === 1) {
            return activeCall;
        }
        if (idx === 2) {
            return myActiveCall;
        }
        if (idx === 3) {
            return isParticipant;
        }
        if (idx === 4) {
            return channelDisplayName;
        }
        return undefined;
    });
};

beforeEach(() => {
    jest.clearAllMocks();
    (useDispatch as unknown as jest.Mock).mockReturnValue(mockDispatch);
});

describe('ChannelHeaderButton visual states', () => {
    it('State 1: Hidden — returns null when pluginEnabled is false', () => {
        setSelectors({pluginEnabled: false});
        const wrapper = shallow(
            <ChannelHeaderButton
                channel={channel}
                currentUserId={currentUserId}
            />,
        );
        expect(wrapper.isEmptyRender()).toBe(true);
    });

    it('State 2 (base): renders Start Call button when no active call', () => {
        setSelectors({pluginEnabled: true, activeCall: undefined, isParticipant: false});
        const wrapper = shallow(
            <ChannelHeaderButton
                channel={channel}
                currentUserId={currentUserId}
            />,
        );
        const btn = wrapper.find('[data-testid="channel-header-call-button"]');
        expect(btn.exists()).toBe(true);
        expect(btn.prop('disabled')).toBe(false);
        const label = wrapper.find('[data-testid="channel-header-call-button-label"]');
        expect(label.text()).toBe('plugin.rtk.channel_header.start_call');
    });

    it('State 3: renders Join Call button when active call exists and user is not participant', () => {
        setSelectors({
            pluginEnabled: true,
            activeCall: {id: 'call1', channelId: 'channel1', participants: ['otherUser'], startAt: 1000000, creatorId: 'otherUser'},
            myActiveCall: null,
            isParticipant: false,
        });
        const wrapper = shallow(
            <ChannelHeaderButton
                channel={channel}
                currentUserId={currentUserId}
            />,
        );
        const label = wrapper.find('[data-testid="channel-header-call-button-label"]');
        expect(label.text()).toBe('plugin.rtk.channel_header.join_call');
    });

    it('State 4: renders In Call (disabled) when user is a participant', () => {
        setSelectors({
            pluginEnabled: true,
            activeCall: {id: 'call1', channelId: 'channel1', participants: [currentUserId], startAt: 1000000, creatorId: 'user1'},
            myActiveCall: {callId: 'call1', channelId: 'channel1', token: 'tok1'},
            isParticipant: true,
        });
        const wrapper = shallow(
            <ChannelHeaderButton
                channel={channel}
                currentUserId={currentUserId}
            />,
        );
        const btn = wrapper.find('[data-testid="channel-header-call-button"]');
        expect(btn.prop('disabled')).toBe(true);
        const label = wrapper.find('[data-testid="channel-header-call-button-label"]');
        expect(label.text()).toBe('plugin.rtk.channel_header.in_call');
    });

    it('State 5: renders error modal when errorMsg is set', async () => {
        const {pluginFetch} = require('client');
        pluginFetch.mockResolvedValueOnce({error: 'Something went wrong'});

        setSelectors({pluginEnabled: true, activeCall: undefined, isParticipant: false});

        // Use mount (not shallow) so React hook state updates flush correctly after async events
        let wrapper: ReturnType<typeof mount>;
        await act(async () => {
            wrapper = mount(
                <ChannelHeaderButton
                    channel={channel}
                    currentUserId={currentUserId}
                />,
            );
        });

        const btn = wrapper!.find('[data-testid="channel-header-call-button"]');
        await act(async () => {
            btn.prop('onClick')?.({} as React.MouseEvent);
        });

        wrapper!.update();
        const errorModal = wrapper!.find('[data-testid="call-error-modal"]');
        expect(errorModal.exists()).toBe(true);
    });
});
