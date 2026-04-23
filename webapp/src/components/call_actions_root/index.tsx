// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {pluginFetch} from 'client';
import React from 'react';
import {useIntl} from 'react-intl';
import {useSelector, useDispatch} from 'react-redux';
import {
    setCallError,
    setMyActiveCall,
    setPendingSwitchCallId,
} from 'redux/calls_slice';
import {playJoinSound} from 'utils/sounds';
import {
    selectCallError,
    selectMyActiveCall,
    selectPendingSwitchCallId,
} from 'redux/selectors';

import SwitchCallModal from 'components/switch_call_modal';

interface JoinCallResponse {
    call: {id: string; channel_id: string};
    token: string;
}

/**
 * Global root component that handles call action modals:
 * - SwitchCallModal: shown when pendingSwitchCallId is set in Redux
 * - Error modal: shown when callError is set in Redux
 *
 * Registered via registry.registerRootComponent so it is always mounted.
 */
const CallActionsRoot = () => {
    const intl = useIntl();
    const dispatch = useDispatch();

    const pendingSwitchCallId = useSelector(selectPendingSwitchCallId);
    const myActiveCall = useSelector(selectMyActiveCall);
    const callError = useSelector(selectCallError);

    const handleSwitchConfirm = async () => {
        if (!pendingSwitchCallId) {
            return;
        }
        dispatch(setPendingSwitchCallId(null));

        if (myActiveCall) {
            pluginFetch(`/api/v1/calls/${myActiveCall.callId}/leave`, {method: 'POST'});
        }

        const result = await pluginFetch<JoinCallResponse>(
            `/api/v1/calls/${pendingSwitchCallId}/token`,
            {method: 'POST'},
        );
        if ('error' in result) {
            dispatch(setCallError(result.error));
            return;
        }
        playJoinSound();
        dispatch(setMyActiveCall({
            callId: result.data.call.id,
            channelId: result.data.call.channel_id,
            token: result.data.token,
        }));
    };

    const handleSwitchCancel = () => {
        dispatch(setPendingSwitchCallId(null));
    };

    const handleErrorClose = () => {
        dispatch(setCallError(null));
    };

    return (
        <>
            <SwitchCallModal
                visible={pendingSwitchCallId !== null}
                onConfirm={handleSwitchConfirm}
                onCancel={handleSwitchCancel}
            />

            {callError && (
                <div
                    className='modal fade in'
                    style={{display: 'block', background: 'rgba(0,0,0,0.5)'}}
                    data-testid='call-error-modal'
                >
                    <div className='modal-dialog'>
                        <div className='modal-content'>
                            <div className='modal-body'>
                                <p>{callError}</p>
                            </div>
                            <div className='modal-footer'>
                                <button
                                    type='button'
                                    className='btn btn-primary'
                                    onClick={handleErrorClose}
                                    data-testid='call-error-modal-close'
                                >
                                    {intl.formatMessage({id: 'plugin.rtk.call_post.error_close'})}
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            )}
        </>
    );
};

export default CallActionsRoot;
