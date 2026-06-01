// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Lightweight i18n. German (de) is primary; English (en) is the fallback.
// Keys map to the labels used by the time-tracking webapp components.

export type StringKey =
    | 'title'
    | 'start'
    | 'stop'
    | 'breakStart'
    | 'breakStop'
    | 'onBreak'
    | 'running'
    | 'idle'
    | 'project'
    | 'noProject'
    | 'note'
    | 'today'
    | 'week'
    | 'hours'
    | 'date'
    | 'startTime'
    | 'endTime'
    | 'breakMinutes'
    | 'net'
    | 'total'
    | 'add'
    | 'edit'
    | 'save'
    | 'delete'
    | 'cancel'
    | 'locked'
    | 'noEntries'
    | 'loading'
    | 'newEntry'
    | 'editEntry'
    | 'invalidRange'
    | 'endRequired'
    | 'adminTitle'
    | 'adminDesc'
    | 'from'
    | 'to'
    | 'user'
    | 'allUsers'
    | 'filterUser'
    | 'exportCsv'
    | 'totalHours'
    | 'people'
    | 'entriesCount';

type Strings = Record<StringKey, string>;

const en: Strings = {
    title: 'Clockwork',
    start: 'Start',
    stop: 'Stop',
    breakStart: 'Break',
    breakStop: 'Resume',
    onBreak: 'On break',
    running: 'Running',
    idle: 'Idle',
    project: 'Project',
    noProject: 'No project',
    note: 'Note',
    today: 'Today',
    week: 'Week',
    hours: 'Hours',
    date: 'Date',
    startTime: 'Start',
    endTime: 'End',
    breakMinutes: 'Break (min)',
    net: 'Net',
    total: 'Total',
    add: 'Add',
    edit: 'Edit',
    save: 'Save',
    delete: 'Delete',
    cancel: 'Cancel',
    locked: 'Locked',
    noEntries: 'No entries',
    loading: 'Loading…',
    newEntry: 'New entry',
    editEntry: 'Edit entry',
    invalidRange: 'End must not be before start',
    endRequired: 'An end time is required',
    adminTitle: 'Team report',
    adminDesc: 'Tracked working time for all users in the selected range.',
    from: 'From',
    to: 'To',
    user: 'User',
    allUsers: 'All users',
    filterUser: 'Filter by user',
    exportCsv: 'Export CSV',
    totalHours: 'Total hours',
    people: 'People',
    entriesCount: 'Entries',
};

const de: Strings = {
    title: 'Clockwork',
    start: 'Start',
    stop: 'Stopp',
    breakStart: 'Pause',
    breakStop: 'Weiter',
    onBreak: 'In Pause',
    running: 'Läuft',
    idle: 'Bereit',
    project: 'Projekt',
    noProject: 'Kein Projekt',
    note: 'Notiz',
    today: 'Heute',
    week: 'Woche',
    hours: 'Stunden',
    date: 'Datum',
    startTime: 'Start',
    endTime: 'Ende',
    breakMinutes: 'Pause (Min)',
    net: 'Netto',
    total: 'Gesamt',
    add: 'Hinzufügen',
    edit: 'Bearbeiten',
    save: 'Speichern',
    delete: 'Löschen',
    cancel: 'Abbrechen',
    locked: 'Gesperrt',
    noEntries: 'Keine Einträge',
    loading: 'Lädt…',
    newEntry: 'Neuer Eintrag',
    editEntry: 'Eintrag bearbeiten',
    invalidRange: 'Ende darf nicht vor dem Start liegen',
    endRequired: 'Eine Endzeit ist erforderlich',
    adminTitle: 'Team-Bericht',
    adminDesc: 'Erfasste Arbeitszeit aller Nutzer im gewählten Zeitraum.',
    from: 'Von',
    to: 'Bis',
    user: 'Nutzer',
    allUsers: 'Alle Nutzer',
    filterUser: 'Nach Nutzer filtern',
    exportCsv: 'CSV exportieren',
    totalHours: 'Gesamtstunden',
    people: 'Personen',
    entriesCount: 'Einträge',
};

const catalogs: Record<string, Strings> = {en, de};

// Default to German per project preference.
let activeLocale = 'de';

// setLocale picks the catalog by the leading language subtag (e.g. "de-DE" ⇒
// "de"). Unknown locales fall back to German.
export function setLocale(locale?: string): void {
    if (!locale) {
        return;
    }
    const lang = locale.toLowerCase().split('-')[0];
    if (catalogs[lang]) {
        activeLocale = lang;
    }
}

// t returns the localized string for key, falling back to English then the key.
export function t(key: StringKey): string {
    const catalog = catalogs[activeLocale] || de;
    return catalog[key] ?? en[key] ?? key;
}
