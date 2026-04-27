// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {pluginFetch} from 'client';
import React, {useEffect, useState} from 'react';
import {useIntl} from 'react-intl';

type AdminStatusResponse = {
    enabled: boolean;
    account_id_via_env: boolean;
    api_token_via_env: boolean;
    cloudflare_account_id: string;
};

// Props passed by Mattermost's admin console to registerAdminConsoleCustomSetting components.
type Props = {
    id: string;
    value: string;
    disabled: boolean;
    onChange: (id: string, value: string) => void;
    setSaveNeeded: () => void;
};

const ENV_VAR_NAMES: Record<string, string> = {
    CloudflareAccountID: 'RTK_ACCOUNT_ID',
    CloudflareAppID: 'RTK_APP_ID',
    CloudflareAPIToken: 'RTK_API_TOKEN',
};

const EnvVarCredentialSetting: React.FC<Props> = ({id, value, disabled, onChange, setSaveNeeded}) => {
    const {formatMessage} = useIntl();
    const [viaEnv, setViaEnv] = useState<boolean | null>(null);

    useEffect(() => {
        pluginFetch<AdminStatusResponse>('/api/v1/config/admin-status').then((result) => {
            if ('data' in result) {
                const keyMap: Record<string, keyof AdminStatusResponse> = {
                    CloudflareAccountID: 'account_id_via_env',
                    CloudflareAppID: 'account_id_via_env',
                    CloudflareAPIToken: 'api_token_via_env',
                };
                const key = keyMap[id] ?? 'account_id_via_env';
                setViaEnv(Boolean(result.data[key]));
            } else {
                setViaEnv(false);
            }
        });
    }, [id]);

    if (viaEnv === null) {
        return (
            <input
                className='form-control'
                type='text'
                value=''
                placeholder={formatMessage({id: 'plugin.rtk.admin.credential.loading'})}
                disabled={true}
                readOnly={true}
            />
        );
    }

    if (viaEnv) {
        const envVarName = ENV_VAR_NAMES[id] ?? id;
        return (
            <input
                className='form-control'
                type='text'
                value={formatMessage({id: 'plugin.rtk.admin.credential.env_var_value'}, {envVar: envVarName})}
                disabled={true}
                readOnly={true}
                style={{color: 'var(--online-indicator)', fontStyle: 'italic'}}
            />
        );
    }

    const isSecret = id === 'CloudflareAPIToken';
    return (
        <input
            className='form-control'
            type={isSecret ? 'password' : 'text'}
            value={value ?? ''}
            disabled={disabled}
            onChange={(e) => {
                onChange(id, e.target.value);
                setSaveNeeded();
            }}
        />
    );
};

export default EnvVarCredentialSetting;
