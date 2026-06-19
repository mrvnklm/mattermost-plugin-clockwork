// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// ensureStyles injects the plugin's stylesheet once. Everything is driven by
// Mattermost theme variables (with fallbacks) so the panel adapts to light/dark
// themes and feels native. Class names are prefixed `tt-`/`tt__` to stay scoped.

const STYLE_ID = 'tt-styles';

const CSS = `
.tt, .tt-admin {
  --tt-mono: 'SFMono-Regular', ui-monospace, Menlo, Consolas, monospace;
  --tt-muted: rgba(var(--center-channel-color-rgb, 61,60,64), .56);
  --tt-faint: rgba(var(--center-channel-color-rgb, 61,60,64), .40);
  --tt-line: rgba(var(--center-channel-color-rgb, 61,60,64), .12);
  --tt-hover: rgba(var(--center-channel-color-rgb, 61,60,64), .06);
  --tt-accent: var(--online-indicator, #06d6a0);
  color: var(--center-channel-color, #3d3c40);
  /* Make native date/number pickers follow the active theme. */
  color-scheme: light dark;
}
.tt { height: 100%; overflow-y: auto; font-size: 14px; }
.tt__err { margin: 12px 16px 0; padding: 8px 12px; border-radius: 8px;
  background: rgba(var(--error-text-rgb, 210,75,78), .12); color: var(--error-text, #d24b4e); font-size: 13px; }
.tt__section { padding: 16px; }
.tt__section + .tt__section { border-top: 1px solid var(--tt-line); }
.tt__label { display: flex; align-items: center; justify-content: space-between;
  font-size: 11px; font-weight: 700; letter-spacing: .06em; text-transform: uppercase;
  color: var(--tt-faint); margin: 0 0 12px; }

.tt-hero { border: 1px solid var(--tt-line); border-radius: 12px; padding: 22px 16px; text-align: center;
  background: radial-gradient(120% 80% at 50% 0%, var(--tt-hover), transparent 70%); transition: border-color .2s ease; }
.tt-hero--run { border-color: color-mix(in srgb, var(--tt-accent) 45%, transparent); }
.tt-hero__state { display: inline-flex; align-items: center; gap: 7px; margin-bottom: 12px;
  font-size: 11px; font-weight: 700; letter-spacing: .08em; text-transform: uppercase; color: var(--tt-muted); }
.tt-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--tt-faint); flex: none; }
.tt-dot--live { background: var(--tt-accent); animation: tt-pulse 1.8s ease-out infinite; }
@keyframes tt-pulse {
  0% { box-shadow: 0 0 0 0 color-mix(in srgb, var(--tt-accent) 55%, transparent); }
  70%, 100% { box-shadow: 0 0 0 9px transparent; } }
.tt-time { font-family: var(--tt-mono); font-size: 44px; line-height: 1.05; font-weight: 600;
  letter-spacing: .01em; font-variant-numeric: tabular-nums; }
.tt-hero__meta { margin-top: 10px; font-size: 13px; color: var(--tt-muted); min-height: 1.2em;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.tt-hero__proj { color: var(--center-channel-color, #3d3c40); font-weight: 600; }
.tt-actions { display: flex; gap: 8px; margin-top: 18px; }
.tt-actions .btn { flex: 1; justify-content: center; }

/* One field style for every control type (text, number, datetime-local, select)
   so heights and borders match everywhere. */
.tt-field { width: 100%; box-sizing: border-box; margin-bottom: 10px; min-height: 38px; padding: 8px 12px;
  border: 1px solid var(--tt-line); border-radius: 8px; background: var(--center-channel-bg, #fff);
  color: var(--center-channel-color, #3d3c40); font: inherit; font-size: 14px; line-height: 1.4; }
.tt-field::placeholder { color: var(--tt-faint); }
.tt-field:focus { outline: none; border-color: var(--button-bg, #1c58d9);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--button-bg, #1c58d9) 22%, transparent); }
.tt-start { width: 100%; justify-content: center; gap: 6px; }

.tt-list { display: flex; flex-direction: column; }
.tt-row { display: flex; align-items: center; gap: 12px; width: 100%; padding: 10px;
  border: 0; border-radius: 8px; background: transparent; color: inherit; text-align: left; font: inherit; }
.tt-row + .tt-row { border-top: 1px solid var(--tt-line); }
.tt-row--btn { cursor: pointer; }
.tt-row--btn:hover { background: var(--tt-hover); }
.tt-row__main { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 3px; }
.tt-row__top { display: flex; align-items: center; gap: 7px; font-size: 13px; }
.tt-chip { font-size: 11px; font-weight: 600; color: var(--tt-muted); background: var(--tt-hover);
  border-radius: 5px; padding: 1px 6px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 130px; }
.tt-note { font-size: 12px; color: var(--tt-muted); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.tt-dur { font-family: var(--tt-mono); font-variant-numeric: tabular-nums; font-size: 13px; flex: none; }
.tt-row--live .tt-dur { color: var(--tt-accent); font-weight: 600; }
.tt-live-tag { font-size: 10px; font-weight: 700; letter-spacing: .06em; color: var(--tt-accent); }

.tt-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.tt-table th { text-align: left; padding: 6px 8px; font-size: 11px; font-weight: 700;
  letter-spacing: .04em; text-transform: uppercase; color: var(--tt-faint); white-space: nowrap; }
.tt-table td { padding: 9px 8px; border-top: 1px solid var(--tt-line); white-space: nowrap; }
.tt-table .tt-num { text-align: right; font-family: var(--tt-mono); font-variant-numeric: tabular-nums; }
.tt-table .tt-date { color: var(--tt-muted); }
.tt-table .tt-act { width: 30px; text-align: right; padding-left: 0; }
.tt-table tbody tr:hover { background: var(--tt-hover); }
.tt-table tfoot td { border-top: 2px solid var(--tt-line); font-weight: 700; }
/* Horizontal scroll fallback so a wide row (e.g. the extra STATUS column with
   the approval workflow on) never clips the hours column in the narrow RHS. */
.tt-tablewrap { width: 100%; overflow-x: auto; }
/* Compact variant for the narrow RHS timesheet so all columns fit without scroll
   in the common case (the full-page admin table keeps the roomier default). */
.tt-table--compact { font-size: 12px; }
.tt-table--compact th { padding: 5px 5px; }
.tt-table--compact td { padding: 7px 5px; }
.tt-table--compact .tt-badge { font-size: 9px; padding: 1px 4px; letter-spacing: 0; }
.tt-iconbtn { display: inline-flex; align-items: center; justify-content: center; width: 26px; height: 26px;
  padding: 0; border: 0; border-radius: 6px; background: transparent; color: var(--tt-faint); cursor: pointer; }
.tt-iconbtn:hover { background: var(--tt-hover); color: var(--center-channel-color, #3d3c40); }
.tt-link { display: inline-flex; align-items: center; gap: 4px; padding: 3px 8px; border: 0; border-radius: 6px;
  background: transparent; color: var(--link-color, var(--button-bg, #1c58d9)); font: inherit; font-size: 12px; font-weight: 600; cursor: pointer; }
.tt-link:hover { background: var(--tt-hover); }
.tt-empty { color: var(--tt-faint); font-size: 13px; padding: 6px 2px; }

.tt-overlay { position: fixed; inset: 0; z-index: 1000; display: flex; align-items: center; justify-content: center;
  background: rgba(0,0,0,.5); }
.tt-modal { width: 380px; max-width: 92vw; border-radius: 12px; padding: 22px;
  background: var(--center-channel-bg, #fff); color: var(--center-channel-color, #3d3c40);
  box-shadow: 0 12px 40px rgba(0,0,0,.35); }
.tt-modal__title { margin: 0 0 16px; font-size: 18px; font-weight: 700; }
.tt-modal__label { display: block; font-size: 12px; font-weight: 600; color: var(--tt-muted); margin: 0 0 4px; }
.tt-modal__row { display: flex; gap: 12px; }
.tt-modal__row > div { flex: 1; }
.tt-modal__foot { display: flex; align-items: center; justify-content: space-between; margin-top: 6px; }
.tt-banner { padding: 8px 12px; border-radius: 8px; font-size: 13px; margin-bottom: 14px; }
.tt-banner--warn { background: var(--tt-hover); color: var(--tt-muted); }
.tt-banner--err { background: rgba(var(--error-text-rgb, 210,75,78), .12); color: var(--error-text, #d24b4e); }

/* admin team report (full-page route) — fill the grid content area */
.tt-admin { box-sizing: border-box; grid-column: 1 / -1; grid-row: 1 / -1; width: 100%; height: 100%;
  overflow-y: auto; padding: 32px; background: var(--center-channel-bg, #fff); font-size: 14px; }
/* product shell (registerProduct) — fills the grid content area; the tab bar is
   fixed and the active report scrolls beneath it. */
.tt-product { box-sizing: border-box; grid-column: 1 / -1; grid-row: 1 / -1; width: 100%; height: 100%;
  display: flex; flex-direction: column; min-height: 0; background: var(--center-channel-bg, #fff); }
.tt-product__tabs { flex: 0 0 auto; display: flex; gap: 6px; padding: 14px 32px 0; border-bottom: 1px solid var(--tt-line); }
.tt-product > .tt-admin { grid-column: auto; grid-row: auto; height: auto; flex: 1 1 auto; min-height: 0; }
.tt-product__title { font-weight: 600; }
.tt-tab { appearance: none; border: none; background: transparent; padding: 8px 14px; font-size: 14px;
  font-weight: 600; color: var(--tt-muted); cursor: pointer; border-bottom: 2px solid transparent; margin-bottom: -1px; }
.tt-tab:hover { color: var(--center-channel-color, #3f4350); }
.tt-tab.is-active { color: var(--button-bg, #1c58d9); border-bottom-color: var(--button-bg, #1c58d9); }
/* Mattermost's global header is transparent and relies on the dark app frame
   that the team sidebar provides; a plugin product has no such frame, so the
   white MM logo would wash out on the light page. Give the header its themed
   background, scoped (via :has) to when the Clockwork product page is mounted so
   the Channels header is never touched. */
#root:has(.tt-product) #global-header,
#root:has(.tt-admin) #global-header { background: var(--global-header-background, #1c2a42); }
.tt-admin__inner { max-width: 1100px; margin: 0 auto; }
.tt-admin__title { margin: 0 0 4px; font-size: 20px; font-weight: 700; }
.tt-admin__sub { margin: 0 0 20px; color: var(--tt-muted); font-size: 13px; }
.tt-admin__bar { display: flex; flex-wrap: wrap; align-items: flex-end; gap: 12px; margin-bottom: 20px; }
.tt-admin__field { display: flex; flex-direction: column; gap: 5px; }
.tt-admin__field > label, .tt-admin__field > .tt-fieldlabel { font-size: 12px; font-weight: 600; color: var(--tt-muted); }
.tt-admin__field .tt-field { width: auto; min-width: 150px; margin: 0; }
.tt-admin__spacer { flex: 1 1 auto; }
.tt-admin__totals { display: flex; flex-wrap: wrap; gap: 12px; margin-bottom: 18px; }
.tt-admin__stat { background: var(--tt-hover); border: 1px solid var(--tt-line); border-radius: 10px; padding: 12px 18px; min-width: 120px; }
.tt-admin__stat b { display: block; font-family: var(--tt-mono); font-size: 22px; font-variant-numeric: tabular-nums; }
.tt-admin__stat span { font-size: 12px; color: var(--tt-muted); }
.tt-admin__empty { padding: 40px; text-align: center; color: var(--tt-faint); }

/* status badges (approval workflow) */
.tt-badge { display: inline-block; font-size: 10px; font-weight: 700; letter-spacing: .04em;
  text-transform: uppercase; padding: 1px 7px; border-radius: 10px; white-space: nowrap; }
.tt-badge--open { background: var(--tt-hover); color: var(--tt-muted); }
.tt-badge--submitted { background: rgba(var(--away-indicator-rgb, 255,188,66), .18); color: var(--away-indicator, #cc8f00); }
.tt-badge--approved { background: rgba(var(--online-indicator-rgb, 6,214,160), .18); color: var(--online-indicator, #06b384); }

/* compact inline action buttons in admin tables */
.tt-actbtns { display: inline-flex; gap: 6px; }
.tt-btn-sm { padding: 3px 9px; border: 1px solid var(--tt-line); border-radius: 6px;
  background: transparent; color: inherit; font: inherit; font-size: 12px; font-weight: 600; cursor: pointer; }
.tt-btn-sm:hover:not(:disabled) { background: var(--tt-hover); }
.tt-btn-sm:disabled { opacity: .5; cursor: default; }

/* week navigation (RHS timesheet) */
.tt-weeknav { display: inline-flex; align-items: center; gap: 2px; }
.tt-weeknav__label { font-size: 11px; color: var(--tt-muted); min-width: 0; }

/* submit-week action row */
.tt-workflow { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; margin-top: 12px; }
.tt-workflow .tt-badge { margin-left: auto; }

/* tabular subsection title inside admin */
.tt-admin__h3 { margin: 24px 0 8px; font-size: 14px; font-weight: 700; }
`;

export function ensureStyles(): void {
    if (typeof document === 'undefined' || document.getElementById(STYLE_ID)) {
        return;
    }
    const el = document.createElement('style');
    el.id = STYLE_ID;
    el.textContent = CSS;
    document.head.appendChild(el);
}
