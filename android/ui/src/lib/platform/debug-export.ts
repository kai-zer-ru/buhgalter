import { registerPlugin } from '@capacitor/core';
import { isNativeApp } from '$lib/platform/native';

export interface DebugExportPlugin {
	saveToDownloads(options: { filename: string; content: string }): Promise<{ path: string }>;
}

export const DebugExport = registerPlugin<DebugExportPlugin>('DebugExport');

export async function saveDebugLogFile(filename: string, content: string): Promise<string> {
	if (isNativeApp()) {
		const result = await DebugExport.saveToDownloads({ filename, content });
		return result.path;
	}
	if (typeof document === 'undefined') {
		throw new Error('download_unavailable');
	}
	const blob = new Blob([content], { type: 'text/plain;charset=utf-8' });
	const url = URL.createObjectURL(blob);
	const anchor = document.createElement('a');
	anchor.href = url;
	anchor.download = filename;
	anchor.click();
	URL.revokeObjectURL(url);
	return filename;
}
