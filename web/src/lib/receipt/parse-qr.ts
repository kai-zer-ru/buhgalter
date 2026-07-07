export type QrScannerHandle = {
	stop: () => Promise<void>;
};

export async function startQrScanner(
	elementId: string,
	onDecoded: (text: string) => void,
	_onError?: (message: string) => void
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
			}
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
			})
			.catch(() => undefined);
	}, 1500);

	return {
		stop: async () => {
			await scanner.stop();
			await scanner.clear();
		}
	};
}
