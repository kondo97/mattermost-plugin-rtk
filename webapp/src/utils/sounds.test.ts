// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {playJoinSound} from './sounds';

const mockStop = jest.fn();
const mockStart = jest.fn();
const mockConnect = jest.fn();
const mockSetValueAtTime = jest.fn();
const mockExponentialRampToValueAtTime = jest.fn();
const mockClose = jest.fn();

const mockGain = {
    connect: mockConnect,
    gain: {
        setValueAtTime: mockSetValueAtTime,
        exponentialRampToValueAtTime: mockExponentialRampToValueAtTime,
    },
};

const mockOscillator = {
    connect: mockConnect,
    type: 'sine' as OscillatorType,
    frequency: {value: 0},
    start: mockStart,
    stop: mockStop,
};

const mockAudioContext = {
    currentTime: 0,
    destination: {},
    createOscillator: jest.fn(() => mockOscillator),
    createGain: jest.fn(() => mockGain),
    close: mockClose,
};

beforeEach(() => {
    jest.clearAllMocks();
    jest.useFakeTimers();
    global.AudioContext = jest.fn(() => mockAudioContext) as unknown as typeof AudioContext;
});

afterEach(() => {
    jest.useRealTimers();
});

describe('playJoinSound', () => {
    it('creates an AudioContext', () => {
        playJoinSound();
        expect(global.AudioContext).toHaveBeenCalledTimes(1);
    });

    it('creates two oscillators for the two tones', () => {
        playJoinSound();
        expect(mockAudioContext.createOscillator).toHaveBeenCalledTimes(2);
    });

    it('starts both oscillators', () => {
        playJoinSound();
        expect(mockStart).toHaveBeenCalledTimes(2);
    });

    it('stops both oscillators', () => {
        playJoinSound();
        expect(mockStop).toHaveBeenCalledTimes(2);
    });

    it('closes the AudioContext after a timeout', () => {
        playJoinSound();
        expect(mockClose).not.toHaveBeenCalled();
        jest.advanceTimersByTime(500);
        expect(mockClose).toHaveBeenCalledTimes(1);
    });

    it('does not throw when AudioContext is unavailable', () => {
        global.AudioContext = undefined as unknown as typeof AudioContext;
        expect(() => playJoinSound()).not.toThrow();
    });
});
