// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Lightweight session info captured once at plugin init (see index.tsx), mirroring
// the setLocale/setTimezone pattern. The product view reads it to decide whether to
// offer the admin "Team report" toggle; the server still re-authorizes every request.

let systemAdmin = false;

export function setIsAdmin(value: boolean): void {
    systemAdmin = value;
}

export function getIsAdmin(): boolean {
    return systemAdmin;
}
