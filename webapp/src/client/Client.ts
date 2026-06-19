// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import manifest from 'manifest';

// TimeEntry mirrors server/store.TimeEntry. All time fields are UTC unix
// MILLISECONDS; a value of 0 means "unset" (end_at: 0 ⇒ running,
// break_started_at: 0 ⇒ not on break).
// EntryStatus mirrors server store.TimeEntry.Status. "locked" is DERIVED
// server-side as (status !== 'open'); keep reading `locked` for the lock banner.
export type EntryStatus = 'open' | 'submitted' | 'approved';

export interface TimeEntry {
    id: string;
    user_id: string;
    start_at: number;
    end_at: number;
    break_seconds: number;
    break_started_at: number;
    project: string;
    description: string;
    status: EntryStatus;
    locked: boolean;
    created_at: number;
    updated_at: number;
}

// Body accepted when creating a manual entry.
export interface CreateEntryBody {
    start_at: number;
    end_at?: number;
    break_seconds?: number;
    project?: string;
    description?: string;
}

// Partial body accepted when updating an entry.
export interface UpdateEntryBody {
    start_at?: number;
    end_at?: number;
    break_seconds?: number;
    project?: string;
    description?: string;
}

// csrfToken reads Mattermost's MMCSRF cookie. Cookie-authenticated mutating
// requests (POST/PUT/DELETE) must echo it as X-CSRF-Token or the server rejects
// them with 401; GET requests are exempt.
function csrfToken(): string {
    const match = (typeof document === 'undefined' ? '' : document.cookie).match(/(?:^|;\s*)MMCSRF=([^;]+)/);
    return match ? decodeURIComponent(match[1]) : '';
}

class Client {
    private baseUrl = `/plugins/${manifest.id}/api/v1`;

    private async request<T>(method: string, path: string, body?: unknown): Promise<T> {
        const headers: Record<string, string> = {
            'Content-Type': 'application/json',

            // Both headers satisfy Mattermost's CSRF check for cookie auth.
            'X-Requested-With': 'XMLHttpRequest',
            'X-CSRF-Token': csrfToken(),
        };
        const options: RequestInit = {
            method,
            headers,
            credentials: 'same-origin',
        };
        if (body !== undefined) {
            options.body = JSON.stringify(body);
        }

        const res = await fetch(this.baseUrl + path, options);

        if (!res.ok) {
            let message = `Request failed with status ${res.status}`;
            try {
                const data = await res.json();
                if (data && typeof data.error === 'string') {
                    message = data.error;
                }
            } catch {
                // body was not JSON; keep the default message
            }
            throw new Error(message);
        }

        // 204 No Content (and any empty body) ⇒ nothing to parse.
        if (res.status === 204) {
            return undefined as unknown as T;
        }
        const text = await res.text();
        if (!text) {
            return undefined as unknown as T;
        }
        return JSON.parse(text) as T;
    }

    // --- live timer ---

    getCurrent(): Promise<{entry: TimeEntry | null}> {
        return this.request('GET', '/timer/current');
    }

    start(project?: string, description?: string): Promise<{entry: TimeEntry}> {
        return this.request('POST', '/timer/start', {project, description});
    }

    stop(): Promise<{entry: TimeEntry}> {
        return this.request('POST', '/timer/stop');
    }

    breakStart(): Promise<{entry: TimeEntry}> {
        return this.request('POST', '/timer/break/start');
    }

    breakStop(): Promise<{entry: TimeEntry}> {
        return this.request('POST', '/timer/break/stop');
    }

    // --- entries ---

    listEntries(from: number, to: number): Promise<{entries: TimeEntry[]}> {
        return this.request('GET', `/entries?from=${from}&to=${to}`);
    }

    createEntry(body: CreateEntryBody): Promise<{entry: TimeEntry}> {
        return this.request('POST', '/entries', body);
    }

    updateEntry(id: string, body: UpdateEntryBody): Promise<{entry: TimeEntry}> {
        return this.request('PUT', `/entries/${encodeURIComponent(id)}`, body);
    }

    deleteEntry(id: string): Promise<void> {
        return this.request('DELETE', `/entries/${encodeURIComponent(id)}`);
    }

    // exportUrl returns the href for the current user's own CSV export (download
    // via the browser so the session cookie authenticates the request).
    exportUrl(from: number, to: number): string {
        return `${this.baseUrl}/reports/export?from=${from}&to=${to}`;
    }

    // --- autocomplete ---

    suggestions(): Promise<{projects: string[]; notes: string[]}> {
        return this.request('GET', '/suggestions');
    }

    // --- config ---

    // config reports server feature flags to the webapp. Requires auth (any
    // logged-in user). approval_enabled gates all workflow UI.
    config(): Promise<{approval_enabled: boolean}> {
        return this.request('GET', '/config');
    }

    // --- approval workflow (only active when approval_enabled; server returns
    // 404 otherwise). from/to are unix MILLISECONDS, matching listEntries. ---

    submitTimesheet(from: number, to: number): Promise<{updated: number}> {
        return this.request('POST', '/timesheet/submit', {from, to});
    }

    withdrawTimesheet(from: number, to: number): Promise<{updated: number}> {
        return this.request('POST', '/timesheet/withdraw', {from, to});
    }

    adminApprove(userId: string, from: number, to: number): Promise<{updated: number}> {
        return this.request('POST', '/admin/approve', {user_id: userId, from, to});
    }

    adminReject(userId: string, from: number, to: number): Promise<{updated: number}> {
        return this.request('POST', '/admin/reject', {user_id: userId, from, to});
    }

    adminReopen(userId: string, from: number, to: number): Promise<{updated: number}> {
        return this.request('POST', '/admin/reopen', {user_id: userId, from, to});
    }

    // --- admin (system admins only; server enforces) ---

    adminEntries(from: number, to: number, userId = ''): Promise<{entries: TimeEntry[]; usernames: Record<string, string>}> {
        const u = userId ? `&user_id=${encodeURIComponent(userId)}` : '';
        return this.request('GET', `/admin/entries?from=${from}&to=${to}${u}`);
    }

    // adminExportUrl returns the href for the admin CSV export (download via the
    // browser so the session cookie authenticates the request).
    adminExportUrl(from: number, to: number, userId = ''): string {
        const u = userId ? `&user_id=${encodeURIComponent(userId)}` : '';
        return `${this.baseUrl}/admin/export?from=${from}&to=${to}${u}`;
    }
}

const client = new Client();
export default client;
