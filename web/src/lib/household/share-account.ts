import { householdKindKey, householdMe } from './membership';
import { get } from 'svelte/store';
import { _ } from 'svelte-i18n';
import {
	ApiError,
	shareAccount,
	unshareAccount,
	type Account,
	type HouseholdMe,
	type ShareAccountResult,
	type ShareLink
} from '$lib/api/client';
import { formatApiError } from '$lib/api/errors';
import { confirm } from '$lib/confirm';
import { toast } from '$lib/toast';

export function canShareAccount(acc: Account, me: HouseholdMe | null | undefined): boolean {
	return Boolean(me?.household) && acc.scope !== 'household' && acc.status === 'active';
}

export function canUnshareAccount(acc: Account, me: HouseholdMe | null | undefined): boolean {
	return acc.scope === 'household' && acc.status === 'active' && me?.role === 'owner';
}

function accountName(id: string, accounts: readonly Account[]): string {
	return accounts.find((acc) => acc.id === id)?.name ?? id;
}

export function formatShareLinkedLines(
	acc: Account,
	links: ShareLink[],
	accounts: readonly Account[],
	t: (key: string, opts?: { values?: Record<string, string> }) => string
): string[] {
	return links.map((link) => {
		const other = accountName(link.id, accounts);
		const values = { from: acc.name, to: other, name: other };
		if (link.kind === 'payment_account') {
			return t('accounts.share.linked.payment', { values });
		}
		if (link.kind === 'auto_topup_source') {
			return t('accounts.share.linked.autoTopup', { values });
		}
		if (link.kind === 'credit_payment_account') {
			return t('accounts.share.linked.creditPay', { values });
		}
		if (link.kind === 'debt_settle_account') {
			return t('accounts.share.linked.debtSettle', { values });
		}
		return t('accounts.share.linked.generic', { values });
	});
}

async function showLinkedPersonal(err: ApiError, acc: Account, accounts: readonly Account[]) {
	const t = get(_);
	const lines = formatShareLinkedLines(acc, err.details?.links ?? [], accounts, t);
	const message = [t('accounts.share.linked.hint'), ...lines].filter(Boolean).join(' ');
	await confirm({
		title: t('accounts.share.linked.title'),
		message,
		acknowledgeOnly: true
	});
}

function toastMoved(result: ShareAccountResult, okKey: string) {
	const t = get(_);
	const count = result.details?.moved?.transactions;
	if (count != null) {
		toast.success(t('accounts.share.moved', { values: { count: String(count) } }));
		return;
	}
	toast.success(t(okKey));
}

export async function promptShareAccount(
	acc: Account,
	accounts: readonly Account[]
): Promise<ShareAccountResult | null> {
	const t = get(_);
	const ok = await confirm({
		title: t('accounts.share'),
		message: t(householdKindKey('accounts.share.confirm', get(householdMe).household?.kind), {
			values: { name: acc.name }
		}),
		confirmLabel: t('accounts.share')
	});
	if (!ok) return null;
	try {
		const result = await shareAccount(acc.id);
		toastMoved(result, 'accounts.share.ok');
		return result;
	} catch (err) {
		if (err instanceof ApiError && err.code === 'ERR_SHARE_LINKED_PERSONAL') {
			await showLinkedPersonal(err, acc, accounts);
			return null;
		}
		toast.error(formatApiError(err));
		return null;
	}
}

export async function promptUnshareAccount(acc: Account): Promise<ShareAccountResult | null> {
	const t = get(_);
	const ownerName = acc.original_owner?.display_name ?? t('household.owner');
	const ok = await confirm({
		title: t('accounts.unshare'),
		message: t('accounts.unshare.confirm', { values: { name: ownerName } }),
		confirmLabel: t('accounts.unshare')
	});
	if (!ok) return null;
	try {
		const result = await unshareAccount(acc.id);
		toastMoved(result, 'accounts.unshare.ok');
		return result;
	} catch (err) {
		toast.error(formatApiError(err));
		return null;
	}
}
