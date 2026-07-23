import { describe, expect, it } from 'vitest';
import {
	accountAutoTopupPath,
	accountChargeFeePath,
	creditActionPath,
	creditCreateStepPath,
	creditNewPath,
	debtNewPath,
	debtSettlePath,
	parseFormReturnPath
} from './form-routes';

describe('form-routes', () => {
	it('parseFormReturnPath rejects external paths', () => {
		expect(parseFormReturnPath('//evil')).toBe('/');
		expect(parseFormReturnPath('/debts')).toBe('/debts');
	});

	it('builds debt new path with direction and debtor', () => {
		expect(debtNewPath({ direction: 'lent', from: '/debts' })).toBe(
			'/debts/new?direction=lent&from=%2Fdebts'
		);
		expect(debtNewPath({ direction: 'borrowed', debtorId: 'd1', from: '/debtors/d1' })).toBe(
			'/debts/new?direction=borrowed&debtor=d1&from=%2Fdebtors%2Fd1'
		);
	});

	it('builds settle and account form paths', () => {
		expect(debtSettlePath('d1', '/debts')).toBe('/debts/d1/settle?from=%2Fdebts');
		expect(accountChargeFeePath('a1', '/accounts')).toBe(
			'/accounts/a1/charge-fee?from=%2Faccounts'
		);
		expect(accountAutoTopupPath('a1')).toBe('/accounts/a1/auto-topup');
	});

	it('builds credit action paths', () => {
		expect(creditActionPath('c1', 'pay', { from: '/credits/c1' })).toBe(
			'/credits/c1/pay?from=%2Fcredits%2Fc1'
		);
		expect(creditActionPath('c1', 'complete')).toBe('/credits/c1/complete');
	});

	it('builds credit create wizard paths', () => {
		expect(creditNewPath('/credits')).toBe('/credits/new?from=%2Fcredits');
		expect(creditCreateStepPath('basics', '/credits')).toBe('/credits/new/basics?from=%2Fcredits');
		expect(creditCreateStepPath('schedule')).toBe('/credits/new/schedule');
	});
});
