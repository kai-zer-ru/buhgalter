const MAX_EDGE_PX = 1600;
const JPEG_QUALITY = 0.82;
/** Already-small JPEGs skip a canvas round-trip. */
const SKIP_BELOW_BYTES = 350_000;
/**
 * Cap the bytes that become base64 in the Capacitor bridge.
 * A multi-megabyte payload rejects the plugin call and the UI shows
 * «server unavailable» without a server log line.
 */
const BRIDGE_SAFE_BYTES = 700_000;

const IMAGE_NAME = /\.(jpe?g|png|webp|heic|heif|bmp|gif)$/i;

/** Photo could not be shrunk enough to cross the native bridge. */
export class ReceiptPhotoError extends Error {
	constructor() {
		super('receipt photo too large');
		this.name = 'ReceiptPhotoError';
	}
}

function shouldCompress(file: File): boolean {
	const smallJpeg =
		file.size <= SKIP_BELOW_BYTES && (file.type === 'image/jpeg' || file.type === 'image/jpg');
	if (smallJpeg) return false;
	if (file.type.startsWith('image/')) return true;
	if (IMAGE_NAME.test(file.name)) return true;
	// Gallery / camera on Android often leaves type empty or octet-stream.
	if ((!file.type || file.type === 'application/octet-stream') && file.size > SKIP_BELOW_BYTES) {
		return true;
	}
	return false;
}

function canUseCanvas(): boolean {
	return typeof document !== 'undefined' && typeof document.createElement === 'function';
}

function loadImageFromFile(file: File): Promise<HTMLImageElement> {
	return new Promise((resolve, reject) => {
		const url = URL.createObjectURL(file);
		const img = new Image();
		img.onload = () => {
			URL.revokeObjectURL(url);
			resolve(img);
		};
		img.onerror = () => {
			URL.revokeObjectURL(url);
			reject(new Error('image decode failed'));
		};
		img.src = url;
	});
}

async function compressOnce(file: File, maxEdge: number, quality: number): Promise<File | null> {
	const img = await loadImageFromFile(file);
	let { width, height } = img;
	const scale = Math.min(1, maxEdge / Math.max(width, height));
	width = Math.max(1, Math.round(width * scale));
	height = Math.max(1, Math.round(height * scale));
	const canvas = document.createElement('canvas');
	canvas.width = width;
	canvas.height = height;
	const ctx = canvas.getContext('2d');
	if (!ctx) return null;
	ctx.drawImage(img, 0, 0, width, height);
	const blob = await new Promise<Blob | null>((resolve) =>
		canvas.toBlob(resolve, 'image/jpeg', quality)
	);
	if (!blob || blob.size === 0) return null;
	const base = file.name.replace(/\.[^.]+$/, '') || 'receipt';
	return new File([blob], `${base}.jpg`, { type: 'image/jpeg' });
}

/** Shrinks receipt photos before native HTTPS upload (Capacitor bridge size limit). */
export async function compressReceiptPhoto(file: File): Promise<File> {
	if (!shouldCompress(file)) return file;
	if (!canUseCanvas()) {
		if (file.size <= BRIDGE_SAFE_BYTES) return file;
		throw new ReceiptPhotoError();
	}
	try {
		let out = await compressOnce(file, MAX_EDGE_PX, JPEG_QUALITY);
		if (out && out.size > BRIDGE_SAFE_BYTES) {
			out = await compressOnce(file, 1280, 0.65);
		}
		if (out && out.size > 0 && out.size <= BRIDGE_SAFE_BYTES) return out;
	} catch {
		// Decode failed (HEIC and similar). Fall through to the size gate.
	}
	if (file.size <= BRIDGE_SAFE_BYTES) return file;
	throw new ReceiptPhotoError();
}
