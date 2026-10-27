import { describe, expect, it } from 'vitest';
import { INVITE_QR_LOGO_RATIO, INVITE_QR_SIZE, inviteQrLogoLayout } from './qr';

describe('inviteQrLogoLayout', () => {
	it('keeps the logo small enough for ECC H', () => {
		const layout = inviteQrLogoLayout();
		expect(INVITE_QR_LOGO_RATIO).toBeLessThanOrEqual(0.25);
		expect(layout.box).toBe(Math.round(INVITE_QR_SIZE * INVITE_QR_LOGO_RATIO));
		expect(layout.inner).toBeGreaterThan(0);
		expect(layout.x + layout.box).toBeLessThanOrEqual(INVITE_QR_SIZE);
		expect(layout.y).toBe(layout.x);
	});
});
