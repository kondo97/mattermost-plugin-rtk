// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {shallow} from 'enzyme';
import React from 'react';
import {useSelector} from 'react-redux';

import ChannelHeaderButton from './index';

// Mock react-redux hooks
jest.mock('react-redux', () => ({
    useSelector: jest.fn(),
    useDispatch: jest.fn(() => jest.fn()),
}));

// Mock the channel-enabled hook to bypass the side-effect fetch.
jest.mock('hooks/use_channel_enabled', () => ({
    useChannelEnabled: jest.fn(),
}));

// eslint-disable-next-line import/first, import/order
import {useChannelEnabled} from 'hooks/use_channel_enabled';

// Mock manifest
jest.mock('manifest', () => ({id: 'com.kondo97.mattermost-plugin-rtk'}));

// Mock react-intl
jest.mock('react-intl', () => ({
    useIntl: () => ({
        formatMessage: ({id}: {id: string}) => id,
    }),
}));

const channel = {id: 'channel1'} as never;
const currentUserId = 'currentUser';

// ChannelHeaderButton reads 4 selectors via useSelector in this order:
// pluginEnabled, activeCall, isParticipant, callLoading.
// useChannelEnabled is mocked separately and is not counted here.
const setSelectors = ({
    pluginEnabled = true,
    activeCall = undefined as object | undefined,
    isParticipant = false,
    callLoading = false,
    channelEnabled = true as boolean | undefined,
} = {}) => {
    const values = [pluginEnabled, activeCall, isParticipant, callLoading];
    let i = 0;
    (useSelector as unknown as jest.Mock).mockImplementation(() => {
        const v = values[i % values.length];
        i++;
        return v;
    });
    (useChannelEnabled as unknown as jest.Mock).mockReturnValue(channelEnabled);
};

beforeEach(() => {
    jest.clearAllMocks();
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

    it('State 1b: Hidden — returns null when channel calls are explicitly disabled', () => {
        setSelectors({pluginEnabled: true, channelEnabled: false});
        const wrapper = shallow(
            <ChannelHeaderButton
                channel={channel}
                currentUserId={currentUserId}
            />,
        );
        expect(wrapper.isEmptyRender()).toBe(true);
    });

    it('State 2: renders Start Call label when no active call', () => {
        setSelectors({pluginEnabled: true, activeCall: undefined, isParticipant: false});
        const wrapper = shallow(
            <ChannelHeaderButton
                channel={channel}
                currentUserId={currentUserId}
            />,
        );
        const btn = wrapper.find('[data-testid="channel-header-call-button"]');
        expect(btn.exists()).toBe(true);
        const label = wrapper.find('[data-testid="channel-header-call-button-label"]');
        expect(label.text()).toBe('plugin.rtk.channel_header.start_call');
    });

    it('State 3: renders Join Call label when active call exists and user is not participant', () => {
        setSelectors({
            pluginEnabled: true,
            activeCall: {id: 'call1', channelId: 'channel1', participants: ['otherUser'], startAt: 1000000, creatorId: 'otherUser'},
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

    it('State 4: renders In Call label when user is a participant', () => {
        setSelectors({
            pluginEnabled: true,
            activeCall: {id: 'call1', channelId: 'channel1', participants: [currentUserId], startAt: 1000000, creatorId: 'user1'},
            isParticipant: true,
        });
        const wrapper = shallow(
            <ChannelHeaderButton
                channel={channel}
                currentUserId={currentUserId}
            />,
        );
        const label = wrapper.find('[data-testid="channel-header-call-button-label"]');
        expect(label.text()).toBe('plugin.rtk.channel_header.in_call');
    });

    it('State 5: renders Starting Call label when loading', () => {
        setSelectors({pluginEnabled: true, activeCall: undefined, isParticipant: false, callLoading: true});
        const wrapper = shallow(
            <ChannelHeaderButton
                channel={channel}
                currentUserId={currentUserId}
            />,
        );
        const label = wrapper.find('[data-testid="channel-header-call-button-label"]');
        expect(label.text()).toBe('plugin.rtk.channel_header.starting_call');
    });
});

