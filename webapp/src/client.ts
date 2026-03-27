// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import manifest from 'manifest';

export type FetchResult<T> = {data: T} | {error: string};

/**
 * Plugin API fetch helper.
 *
 * - Always resolves (never throws to caller).
 * - Returns { data: T } on success, { error: string } on failure.
 * - Generic error messages only — no raw server details surfaced to users (SEC-U3-03).
 * - JWT tokens and credentials MUST NOT be logged (SEC-U3-01).
 */
export async function pluginFetch<T>(
    path: string,
    options?: RequestInit,
): Promise<FetchResult<T>> {
    try {
        const resp = await fetch(`/plugins/${manifest.id}${path}`, {
            headers: {
                'Content-Type': 'application/json',
                'X-Requested-With': 'XMLHttpRequest',
            },
            ...options,
        });

        if (!resp.ok) {
            // Log status/path only — no response body that might contain sensitive data
            console.error(`[rtk-plugin] API error ${resp.status} on ${path}`); // eslint-disable-line no-console
            return {error: 'An error occurred. Please try again.'};
        }

        const data = await resp.json() as T;
        return {data};
    } catch (err) {
        console.error('[rtk-plugin] network error on', path, err); // eslint-disable-line no-console
        return {error: 'A network error occurred. Please try again.'};
    }
}
