/** One part for native OkHttp multipart (SslTrust plugin). */
export type NativeMultipartPart =
	| { name: string; value: string }
	| { name: string; filename: string; mime: string; base64: string };

function uint8ToBase64(bytes: Uint8Array): string {
	// Spread of 32k+ code units throws on Android WebView ("too many arguments")
	// and the upload never leaves the phone.
	const chunk = 0x2000;
	let binary = '';
	for (let i = 0; i < bytes.length; i += chunk) {
		const slice = bytes.subarray(i, i + chunk);
		let part = '';
		for (let j = 0; j < slice.length; j++) part += String.fromCharCode(slice[j]);
		binary += part;
	}
	return btoa(binary);
}

export async function formDataToNativeMultipart(form: FormData): Promise<NativeMultipartPart[]> {
	const parts: NativeMultipartPart[] = [];
	for (const [name, entry] of form.entries()) {
		if (entry instanceof File) {
			const buf = await entry.arrayBuffer();
			parts.push({
				name,
				filename: entry.name || 'upload.bin',
				mime: entry.type || 'application/octet-stream',
				base64: uint8ToBase64(new Uint8Array(buf))
			});
		} else {
			parts.push({ name, value: String(entry) });
		}
	}
	return parts;
}
