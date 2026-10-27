import QRCode from 'qrcode';

export const INVITE_QR_SIZE = 480;
/** Logo box as a fraction of QR width; stays well under ECC H recovery (~30% of modules). */
export const INVITE_QR_LOGO_RATIO = 0.2;
const LOGO_SRC = '/icon-192.png';

export function inviteQrLogoLayout(size = INVITE_QR_SIZE) {
	const box = Math.round(size * INVITE_QR_LOGO_RATIO);
	const pad = Math.max(4, Math.round(size * 0.016));
	return {
		box,
		pad,
		x: Math.round((size - box) / 2),
		y: Math.round((size - box) / 2),
		inner: box - pad * 2
	};
}

export async function inviteQrDataUrl(url: string): Promise<string> {
	const qr = await QRCode.toDataURL(url, {
		margin: 2,
		width: INVITE_QR_SIZE,
		errorCorrectionLevel: 'H',
		color: { dark: '#111827', light: '#ffffff' }
	});
	if (typeof document === 'undefined') return qr;
	try {
		return await overlayLogo(qr);
	} catch {
		return qr;
	}
}

function loadImage(src: string): Promise<HTMLImageElement> {
	return new Promise((resolve, reject) => {
		const img = new Image();
		img.decoding = 'async';
		img.onload = () => resolve(img);
		img.onerror = () => reject(new Error('qr logo image failed'));
		img.src = src;
	});
}

async function overlayLogo(qrDataUrl: string): Promise<string> {
	const canvas = document.createElement('canvas');
	canvas.width = INVITE_QR_SIZE;
	canvas.height = INVITE_QR_SIZE;
	const ctx = canvas.getContext('2d');
	if (!ctx) return qrDataUrl;

	const qrImg = await loadImage(qrDataUrl);
	ctx.drawImage(qrImg, 0, 0, INVITE_QR_SIZE, INVITE_QR_SIZE);

	const logoImg = await loadImage(LOGO_SRC);
	const { box, pad, x, y, inner } = inviteQrLogoLayout();
	const radius = Math.max(6, Math.round(box * 0.16));
	ctx.beginPath();
	if (typeof ctx.roundRect === 'function') {
		ctx.roundRect(x, y, box, box, radius);
	} else {
		ctx.rect(x, y, box, box);
	}
	ctx.fillStyle = '#ffffff';
	ctx.fill();
	ctx.drawImage(logoImg, x + pad, y + pad, inner, inner);
	return canvas.toDataURL('image/png');
}
