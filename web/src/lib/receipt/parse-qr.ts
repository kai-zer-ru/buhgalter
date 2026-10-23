export type QrScannerHandle = {
	stop: () => Promise<void>;
};

export async function startQrScanner(
	elementId: string,
	onDecoded: (text: string) => void
): Promise<QrScannerHandle> {
	const { Html5Qrcode } = await import('html5-qrcode');
	const scanner = new Html5Qrcode(elementId, {
		verbose: false,
		useBarCodeDetectorIfSupported: true
	});
	await scanner.start(
		{
			facingMode: 'environment',
			width: { ideal: 1920 },
			height: { ideal: 1080 }
		},
		{
			fps: 10,
			qrbox: (viewfinderWidth, viewfinderHeight) => {
				const size = Math.floor(Math.min(viewfinderWidth, viewfinderHeight) * 0.75);
				return { width: size, height: size };
			},
			disableFlip: false,
			videoConstraints: {
				facingMode: 'environment',
				focusMode: 'continuous',
				width: { ideal: 1920 },
				height: { ideal: 1080 }
			} as MediaTrackConstraints
		},
		(decoded) => onDecoded(decoded),
		() => {
			/* ignore frame errors */
		}
	);

	window.setTimeout(() => {
		scanner
			.applyVideoConstraints({
				focusMode: 'continuous',
				advanced: [{ zoom: 2 }]
			} as unknown as MediaTrackConstraints)
			.catch(() => undefined);
	}, 1500);

	return {
		stop: async () => {
			await scanner.stop();
			await scanner.clear();
		}
	};
}

type BarcodeHit = { rawValue?: string };
type BarcodeDetectorLike = { detect: (source: ImageBitmap) => Promise<BarcodeHit[]> };

/** Read a QR from a gallery photo on the device. Null when the picture has no readable QR. */
export async function decodeQrFromImageFile(file: File): Promise<string | null> {
	try {
		const fromDetector = await decodeWithBarcodeDetector(file);
		if (fromDetector) return fromDetector;
	} catch {
		/* try html5-qrcode */
	}
	try {
		return await decodeWithHtml5(file);
	} catch {
		return null;
	}
}

async function decodeWithBarcodeDetector(file: File): Promise<string | null> {
	const Ctor = (
		globalThis as { BarcodeDetector?: new (opts?: { formats?: string[] }) => BarcodeDetectorLike }
	).BarcodeDetector;
	if (!Ctor || typeof createImageBitmap !== 'function') return null;
	const detector = new Ctor({ formats: ['qr_code'] });
	const bitmap = await createImageBitmap(file);
	try {
		const codes = await detector.detect(bitmap);
		const text = codes.find((code) => code.rawValue)?.rawValue?.trim();
		return text || null;
	} finally {
		bitmap.close();
	}
}

async function decodeWithHtml5(file: File): Promise<string | null> {
	if (typeof document === 'undefined') return null;
	const { Html5Qrcode } = await import('html5-qrcode');
	const host = document.createElement('div');
	host.id = `qr-file-${Date.now()}`;
	host.hidden = true;
	document.body.appendChild(host);
	try {
		const scanner = new Html5Qrcode(host.id, { verbose: false });
		const text = (await scanner.scanFile(file, false)).trim();
		return text || null;
	} finally {
		host.remove();
	}
}
