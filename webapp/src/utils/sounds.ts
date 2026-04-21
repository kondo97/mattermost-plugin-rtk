// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

/**
 * Play a short two-tone ascending sound cue when the local user joins a call.
 * Uses the Web Audio API so no audio asset is required.
 * Errors are silently ignored (e.g. in test environments where AudioContext
 * is unavailable, or when the browser blocks audio before a user gesture).
 */
export function playJoinSound(): void {
    try {
        const ctx = new AudioContext();

        const playTone = (frequency: number, startOffset: number, duration: number) => {
            const osc = ctx.createOscillator();
            const gain = ctx.createGain();
            osc.connect(gain);
            gain.connect(ctx.destination);

            osc.type = 'sine';
            osc.frequency.value = frequency;

            const startTime = ctx.currentTime + startOffset;
            gain.gain.setValueAtTime(0.25, startTime);
            gain.gain.exponentialRampToValueAtTime(0.001, startTime + duration);

            osc.start(startTime);
            osc.stop(startTime + duration);
        };

        // Ascending two-tone: A5 (880 Hz) then C#6 (1108 Hz)
        playTone(880, 0, 0.15);
        playTone(1108, 0.15, 0.15);

        // Release the AudioContext after the sounds finish
        setTimeout(() => ctx.close(), 500);
    } catch {
        // Silently ignore — AudioContext may be unavailable in test environments
        // or blocked by browser autoplay policy
    }
}
