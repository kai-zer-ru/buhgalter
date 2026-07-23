import { get } from 'svelte/store';
import { gotoReplace } from '$lib/android/form-nav';
import {
	creditCreateStepPath,
	type CreditCreateStep,
	parseFormReturnPath
} from '$lib/android/form-routes';
import { beginCreditCreate, clearCreditCreate, creditCreateDraft } from './create-draft';

const STEPS: CreditCreateStep[] = ['basics', 'options', 'schedule'];

export function creditCreateReturnTo(fromRaw: string | null): string {
	return parseFormReturnPath(fromRaw, '/credits');
}

export function ensureCreditCreateDraft(tz: string) {
	if (!get(creditCreateDraft)) beginCreditCreate(tz);
}

export function abandonCreditCreate(returnTo: string) {
	clearCreditCreate();
	void gotoReplace(returnTo);
}

export function goCreditCreateStep(step: CreditCreateStep, fromRaw: string | null) {
	const from = creditCreateReturnTo(fromRaw);
	void gotoReplace(creditCreateStepPath(step, from));
}

export function nextCreditCreateStep(
	current: CreditCreateStep,
	fromRaw: string | null
): CreditCreateStep | null {
	const i = STEPS.indexOf(current);
	if (i < 0 || i >= STEPS.length - 1) return null;
	const next = STEPS[i + 1];
	goCreditCreateStep(next, fromRaw);
	return next;
}

export function prevCreditCreateStep(
	current: CreditCreateStep,
	fromRaw: string | null,
	returnTo: string
) {
	const i = STEPS.indexOf(current);
	if (i <= 0) {
		abandonCreditCreate(returnTo);
		return;
	}
	goCreditCreateStep(STEPS[i - 1], fromRaw);
}
