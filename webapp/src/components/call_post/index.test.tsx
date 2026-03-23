// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {act} from 'react-dom/test-utils';
import {shallow, mount} from 'enzyme';
import {useSelector, useDispatch} from 'react-redux';

import CallPost from './index';

jest.mock('react-redux', () => ({
    useSelector: jest.fn(),
    useDispatch: jest.fn(),
}));

jest.mock('client', () => ({pluginFetch: jest.fn()}));
jest.mock('manifest', () => ({id: 'com.mattermost.plugin-rtk'}));
jest.mock('react-intl', () => ({
    useIntl: () => ({formatMessage: ({id}: {id: string}) => id}),
}));
jest.mock('components/switch_call_modal', () => () => null);
jest.mock('utils/call_tab', () => ({
    buildCallTabUrl: jest.fn(() => 'mock-url'),
    getChannelDisplayName: jest.fn(() => 'general'),
}));

const mockDispatch = jest.fn();

const makePost = (overrides: object = {}) => ({
    id: 'post1',
    props: {
        call_id: 'call1',
        channel_id: 'channel1',
        creator_id: 'user1',
        start_at: 1000000,
        end_at: 0,
        participants: ['user1', 'user2'],
        ...overrides,
    },
});

const setSelectors = (liveCall: object | undefined, myActiveCall: object | null, channelName = 'general') => {
    (useSelector as jest.Mock).mockImplementation(() => {
        // Use modulo so re-renders (calls 3-5, 6-8, …) return the same values.
        const idx = (useSelector as jest.Mock).mock.calls.length % 3;
        if (idx === 1) { return liveCall; }
        if (idx === 2) { return myActiveCall; }
        return channelName;
    });
};

beforeEach(() => {
    jest.clearAllMocks();
    (useDispatch as jest.Mock).mockReturnValue(mockDispatch);
});

describe('CallPost', () => {
    it('renders active state when end_at is 0 and no live Redux data', () => {
        setSelectors(undefined, null);
        const wrapper = shallow(<CallPost post={makePost()} />);
        expect(wrapper.find('[data-testid="call-post"]').exists()).toBe(true);
        // CallPostActive should be rendered (not CallPostEnded)
        expect(wrapper.find('CallPostActive').exists()).toBe(true);
        expect(wrapper.find('CallPostEnded').exists()).toBe(false);
    });

    it('renders ended state when end_at > 0', () => {
        setSelectors(undefined, null);
        const wrapper = shallow(<CallPost post={makePost({end_at: 2000000})} />);
        expect(wrapper.find('CallPostEnded').exists()).toBe(true);
        expect(wrapper.find('CallPostActive').exists()).toBe(false);
    });

    it('renders ended state when live Redux call has endAt > 0', () => {
        const liveCall = {
            id: 'call1', channelId: 'channel1', creatorId: 'user1',
            participants: ['user1'], startAt: 1000000, endAt: 2000000,
        };
        setSelectors(liveCall, null);
        const wrapper = shallow(<CallPost post={makePost()} />);
        expect(wrapper.find('CallPostEnded').exists()).toBe(true);
    });

    it('passes isAlreadyInCall=true to CallPostActive when myActiveCall matches', () => {
        setSelectors(undefined, {callId: 'call1', channelId: 'channel1', token: 'tok'});
        const wrapper = shallow(<CallPost post={makePost()} />);
        const active = wrapper.find('CallPostActive');
        expect(active.prop('isAlreadyInCall')).toBe(true);
    });

    it('passes isAlreadyInCall=false when myActiveCall is null', () => {
        setSelectors(undefined, null);
        const wrapper = shallow(<CallPost post={makePost()} />);
        const active = wrapper.find('CallPostActive');
        expect(active.prop('isAlreadyInCall')).toBe(false);
    });

    it('passes isAlreadyInCall=false when myActiveCall is a different call', () => {
        setSelectors(undefined, {callId: 'call99', channelId: 'channel99', token: 'tok'});
        const wrapper = shallow(<CallPost post={makePost()} />);
        const active = wrapper.find('CallPostActive');
        expect(active.prop('isAlreadyInCall')).toBe(false);
    });

    it('renders error modal when errorMsg is set after API failure', async () => {
        const {pluginFetch} = require('client');
        pluginFetch.mockResolvedValueOnce({error: 'Something went wrong'});
        setSelectors(undefined, null);

        // Use mount (not shallow) so React hook state updates flush correctly after async events
        let wrapper: ReturnType<typeof mount>;
        await act(async () => {
            wrapper = mount(<CallPost post={makePost()} />);
        });

        // Trigger onJoin via the CallPostActive prop
        const active = wrapper!.find('CallPostActive');
        await act(async () => {
            active.prop('onJoin')?.();
        });

        wrapper!.update();
        expect(wrapper!.find('[data-testid="call-post-error-modal"]').exists()).toBe(true);
    });

    it('uses live Redux participants over post.props participants', () => {
        const liveCall = {
            id: 'call1', channelId: 'channel1', creatorId: 'user1',
            participants: ['user1', 'user2', 'user3'], startAt: 1000000,
        };
        setSelectors(liveCall, null);
        const wrapper = shallow(<CallPost post={makePost({participants: ['user1']})} />);
        const active = wrapper.find('CallPostActive');
        expect((active.prop('participants') as string[]).length).toBe(3);
    });
});
