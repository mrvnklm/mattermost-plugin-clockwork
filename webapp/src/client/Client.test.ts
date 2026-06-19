/**
 * @jest-environment jsdom
 */

// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import Client from './Client';

// manifest.id is 'com.vsjwl.mm-time-tracking' (see manifest.ts); the base URL
// is /plugins/<id>/api/v1.
const BASE = '/plugins/com.vsjwl.mm-time-tracking/api/v1';

type FetchCall = {url: string; init: RequestInit};

let calls: FetchCall[];

function mockFetch(response: {ok?: boolean; status?: number; body?: unknown; text?: string}) {
    const ok = response.ok ?? true;
    const status = response.status ?? (ok ? 200 : 400);
    const text = response.text ?? (response.body === undefined ? '' : JSON.stringify(response.body));
    calls = [];
    global.fetch = jest.fn((url: string, init: RequestInit) => {
        calls.push({url, init});
        return Promise.resolve({
            ok,
            status,
            text: () => Promise.resolve(text),
            json: () => Promise.resolve(text ? JSON.parse(text) : undefined),
        } as unknown as Response);
    }) as unknown as typeof fetch;
}

beforeEach(() => {
    calls = [];

    // Provide a CSRF cookie for mutating-request tests. jsdom's document.cookie
    // is a real store; set the value (URL-encoded space ⇒ decoded by the client).
    document.cookie = 'MMCSRF=tok%20en';
});

describe('request plumbing', () => {
    it('sends CSRF + XHR headers and same-origin credentials on POST', async () => {
        mockFetch({body: {entry: {}}});
        await Client.start('proj', 'note');

        expect(calls).toHaveLength(1);
        const {url, init} = calls[0];
        expect(url).toBe(`${BASE}/timer/start`);
        expect(init.method).toBe('POST');
        expect(init.credentials).toBe('same-origin');

        const headers = init.headers as Record<string, string>;
        expect(headers['X-CSRF-Token']).toBe('tok en'); // URL-decoded
        expect(headers['X-Requested-With']).toBe('XMLHttpRequest');
        expect(headers['Content-Type']).toBe('application/json');
        expect(init.body).toBe(JSON.stringify({project: 'proj', description: 'note'}));
    });

    it('omits the body on a bodyless POST', async () => {
        mockFetch({body: {entry: {}}});
        await Client.stop();
        expect(calls[0].init.body).toBeUndefined();
    });

    it('parses a JSON error body into the thrown message', async () => {
        mockFetch({ok: false, status: 423, body: {error: 'entry is locked'}});
        await expect(Client.deleteEntry('id1')).rejects.toThrow('entry is locked');
    });

    it('falls back to a status message when the error body is not JSON', async () => {
        mockFetch({ok: false, status: 500, text: '<html>oops</html>'});
        await expect(Client.getCurrent()).rejects.toThrow('Request failed with status 500');
    });

    it('treats 204 and empty bodies as undefined', async () => {
        mockFetch({status: 204, text: ''});
        await expect(Client.deleteEntry('id1')).resolves.toBeUndefined();
    });

    it('URL-encodes path segments', async () => {
        mockFetch({body: {entry: {}}});
        await Client.updateEntry('a/b c', {project: 'p'});
        expect(calls[0].url).toBe(`${BASE}/entries/a%2Fb%20c`);
        expect(calls[0].init.method).toBe('PUT');
    });
});

describe('config + workflow endpoints', () => {
    it('GET /config', async () => {
        mockFetch({body: {approval_enabled: true}});
        const res = await Client.config();
        expect(calls[0].url).toBe(`${BASE}/config`);
        expect(calls[0].init.method).toBe('GET');
        expect(res.approval_enabled).toBe(true);
    });

    it('POST /timesheet/submit with from/to body', async () => {
        mockFetch({body: {updated: 3}});
        const res = await Client.submitTimesheet(100, 200);
        expect(calls[0].url).toBe(`${BASE}/timesheet/submit`);
        expect(calls[0].init.body).toBe(JSON.stringify({from: 100, to: 200}));
        expect(res.updated).toBe(3);
    });

    it('POST /timesheet/withdraw', async () => {
        mockFetch({body: {updated: 1}});
        await Client.withdrawTimesheet(5, 6);
        expect(calls[0].url).toBe(`${BASE}/timesheet/withdraw`);
        expect(calls[0].init.body).toBe(JSON.stringify({from: 5, to: 6}));
    });

    it('admin approve/reject/reopen send user_id + range', async () => {
        mockFetch({body: {updated: 2}});
        await Client.adminApprove('user9', 10, 20);
        expect(calls[0].url).toBe(`${BASE}/admin/approve`);
        expect(calls[0].init.body).toBe(JSON.stringify({user_id: 'user9', from: 10, to: 20}));

        mockFetch({body: {updated: 0}});
        await Client.adminReject('user9', 10, 20);
        expect(calls[0].url).toBe(`${BASE}/admin/reject`);

        mockFetch({body: {updated: 0}});
        await Client.adminReopen('user9', 10, 20);
        expect(calls[0].url).toBe(`${BASE}/admin/reopen`);
        expect(calls[0].init.body).toBe(JSON.stringify({user_id: 'user9', from: 10, to: 20}));
    });
});

describe('admin entries + export URL pass user_id', () => {
    it('adminEntries appends user_id only when set', async () => {
        mockFetch({body: {entries: [], usernames: {}}});
        await Client.adminEntries(1, 2);
        expect(calls[0].url).toBe(`${BASE}/admin/entries?from=1&to=2`);

        mockFetch({body: {entries: [], usernames: {}}});
        await Client.adminEntries(1, 2, 'u42');
        expect(calls[0].url).toBe(`${BASE}/admin/entries?from=1&to=2&user_id=u42`);
    });

    it('adminExportUrl includes user_id when narrowed (export-bug fix)', () => {
        expect(Client.adminExportUrl(1, 2)).toBe(`${BASE}/admin/export?from=1&to=2`);
        expect(Client.adminExportUrl(1, 2, 'u42')).toBe(`${BASE}/admin/export?from=1&to=2&user_id=u42`);
    });

    it('exportUrl points at the current user own report export', () => {
        expect(Client.exportUrl(1, 2)).toBe(`${BASE}/reports/export?from=1&to=2`);
    });
});
