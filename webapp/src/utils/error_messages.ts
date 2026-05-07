// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import enMessages from '../../i18n/en.json';
import jaMessages from '../../i18n/ja.json';

type LocaleMessages = Record<string, string>;

const locales: Record<string, LocaleMessages> = {
    en: enMessages as LocaleMessages,
    ja: jaMessages as LocaleMessages,
};

// Maps server-side error codes (returned in the JSON error body as `code`)
// to the i18n message id displayed in the call error modal.
const ERROR_CODE_MESSAGE_IDS: Record<string, string> = {
    calls_disabled: 'plugin.rtk.error.calls_disabled',
};

function detectLocale(): string {
    const lang = (typeof navigator !== 'undefined' && navigator.language) || 'en';
    const primary = lang.toLowerCase().split('-')[0];
    return primary in locales ? primary : 'en';
}

// getCallErrorMessage resolves a localised, user-friendly error message for
// the call error modal.
//
// - When the server returns a known `code`, the corresponding localised
//   message is returned (using the user's browser locale, falling back to en).
// - When the code is unknown or absent, `fallback` (the generic message
//   produced by pluginFetch, or the raw server error string) is returned.
//
// This is invoked from non-component code (the click handler registered
// via registerCallButtonAction in index.tsx) where the React-Intl context
// is unavailable, so we look messages up directly from the bundled JSON.
export function getCallErrorMessage(code: string | undefined, fallback: string): string {
    if (!code) {
        return fallback;
    }
    const messageId = ERROR_CODE_MESSAGE_IDS[code];
    if (!messageId) {
        return fallback;
    }
    const locale = detectLocale();
    return locales[locale]?.[messageId] || locales.en[messageId] || fallback;
}
