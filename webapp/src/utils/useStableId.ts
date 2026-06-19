// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {useState} from 'react';

let counter = 0;

// useStableId returns a process-unique, render-stable id. It replaces React 18's
// useId(), which is NOT available to Mattermost plugins: the host webapp injects
// its own React (v17 on current Mattermost), so calling useId() throws
// "useId is not a function" and crashes the component. Used to wire
// <label htmlFor>/aria-* without id collisions across multiple instances.
export function useStableId(prefix = 'cw'): string {
    const [id] = useState(() => `${prefix}-${++counter}`);
    return id;
}
