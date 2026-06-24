/** Fallback when Intl.supportedValuesOf is unavailable (SSR / old browsers). */
export const TIMEZONE_FALLBACK = [
	'UTC',
	'Europe/Moscow',
	'Europe/London',
	'America/New_York'
] as const;
