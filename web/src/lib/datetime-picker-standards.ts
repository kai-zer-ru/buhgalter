/**
 * Standard DateTimePicker props for operation datetime fields.
 * See docs/date-time-display.md
 */
export const operationDatetimePickerCreate = {
	timeMode: 'optional' as const,
	defaultTime: 'now' as const
};

export const operationDatetimePickerEdit = {
	timeMode: 'optional' as const,
	defaultTime: 'preserve' as const
};

/** Default local time for credit auto-debit (user timezone). */
export const defaultAutoDebitTimeLocal = '08:00';

/** Date-only fields (filters, due dates, credit issue date, …). */
export const dateOnlyPicker = {
	timeMode: 'hidden' as const
};
