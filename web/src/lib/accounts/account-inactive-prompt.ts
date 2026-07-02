import { get } from 'svelte/store';
import { _ } from 'svelte-i18n';
import type { Account } from '$lib/api/client';
import { archiveAccount, deleteAccount } from '$lib/api/client';
import { confirmAccountTransfer } from '$lib/accounts/account-transfer-confirm';
import {
	defaultTransferAccountId,
	needsBalanceTransfer,
	transferAccountOptions
} from '$lib/transfer-accounts';
import { isCreditCard, isCreditCardFullyPaid } from '$lib/credit-card';

export type PromptAccountInactiveParams = {
	acc: Account;
	activeAccounts: Account[];
};

async function promptCreditCardInactive(
	acc: Account,
	action: 'archive' | 'delete'
): Promise<{ ok: boolean; transferToAccountId?: string }> {
	const { confirm } = await import('$lib/confirm');
	if (!isCreditCardFullyPaid(acc)) {
		await confirm({
			message: get(_)('accounts.confirm.creditCardNotFullyPaid'),
			acknowledgeOnly: true
		});
		return { ok: false };
	}
	const ok = await confirm({
		message: get(_)(action === 'archive' ? 'accounts.confirm.archive' : 'accounts.confirm.delete'),
		confirmLabel: get(_)(action === 'archive' ? 'accounts.action.archive' : 'common.delete'),
		danger: action === 'delete'
	});
	return { ok };
}

async function promptWithBalanceTransfer(opts: {
	acc: Account;
	activeAccounts: Account[];
	message: string;
	balanceMessageBeforeKey: string;
	balanceMessageAfterKey: string;
	confirmLabel: string;
	danger: boolean;
}): Promise<{ ok: boolean; transferToAccountId?: string }> {
	const transferOptions = transferAccountOptions(opts.activeAccounts, opts.acc.id);
	const result = await confirmAccountTransfer({
		message: opts.message,
		needsTransfer: true,
		balanceMessageBefore: get(_)(opts.balanceMessageBeforeKey),
		balanceMessageAfter: get(_)(opts.balanceMessageAfterKey),
		balanceDisplay: opts.acc.balance_display,
		transferLabel: get(_)('accounts.confirm.transferTo'),
		noTargetsMessage:
			transferOptions.length === 0 ? get(_)('accounts.confirm.inactiveNoTargets') : '',
		transferOptions,
		initialTransferAccountId: defaultTransferAccountId(opts.activeAccounts, opts.acc.id),
		confirmLabel: opts.confirmLabel,
		danger: opts.danger
	});
	return result;
}

export async function promptArchiveAccount({
	acc,
	activeAccounts
}: PromptAccountInactiveParams): Promise<{ ok: boolean; transferToAccountId?: string }> {
	if (isCreditCard(acc)) {
		return promptCreditCardInactive(acc, 'archive');
	}
	if (!needsBalanceTransfer(acc)) {
		const { confirm } = await import('$lib/confirm');
		const ok = await confirm({
			message: get(_)('accounts.confirm.archive'),
			confirmLabel: get(_)('accounts.action.archive'),
			danger: false
		});
		return { ok };
	}
	return promptWithBalanceTransfer({
		acc,
		activeAccounts,
		message: get(_)('accounts.confirm.archive'),
		balanceMessageBeforeKey: 'accounts.confirm.archiveWithBalance.before',
		balanceMessageAfterKey: 'accounts.confirm.archiveWithBalance.after',
		confirmLabel: get(_)('accounts.action.archive'),
		danger: false
	});
}

export async function promptDeleteAccount({
	acc,
	activeAccounts
}: PromptAccountInactiveParams): Promise<{ ok: boolean; transferToAccountId?: string }> {
	if (isCreditCard(acc)) {
		return promptCreditCardInactive(acc, 'delete');
	}
	if (!needsBalanceTransfer(acc)) {
		const { confirm } = await import('$lib/confirm');
		const ok = await confirm({
			message: get(_)('accounts.confirm.delete'),
			confirmLabel: get(_)('common.delete'),
			danger: true
		});
		return { ok };
	}
	return promptWithBalanceTransfer({
		acc,
		activeAccounts,
		message: get(_)('accounts.confirm.delete'),
		balanceMessageBeforeKey: 'accounts.confirm.deleteWithBalance.before',
		balanceMessageAfterKey: 'accounts.confirm.deleteWithBalance.after',
		confirmLabel: get(_)('common.delete'),
		danger: true
	});
}

export async function executeArchiveAccount(
	acc: Account,
	transferToAccountId?: string
): Promise<Account> {
	return archiveAccount(acc.id, transferToAccountId);
}

export async function executeDeleteAccount(
	acc: Account,
	transferToAccountId?: string
): Promise<void> {
	await deleteAccount(acc.id, transferToAccountId);
}
