import { registerPlugin } from '@capacitor/core';
import { isNativeApp } from '$lib/platform/native';
import type { WidgetSnapshot } from './snapshot';

export type WidgetBridgePublishPayload = {
	baseUrl: string;
	token: string;
	lockEnabled: boolean;
	snapshot: WidgetSnapshot;
};

type WidgetBridgePlugin = {
	publish(options: WidgetBridgePublishPayload): Promise<void>;
	setLockEnabled(options: { lockEnabled: boolean }): Promise<void>;
	clear(): Promise<void>;
};

const WidgetBridge = registerPlugin<WidgetBridgePlugin>('WidgetBridge');

export async function publishWidgetBridge(payload: WidgetBridgePublishPayload): Promise<void> {
	if (!isNativeApp()) return;
	try {
		await WidgetBridge.publish(payload);
	} catch {
		// widgets must not break the app
	}
}

export async function setWidgetLockEnabled(lockEnabled: boolean): Promise<void> {
	if (!isNativeApp()) return;
	try {
		await WidgetBridge.setLockEnabled({ lockEnabled });
	} catch {
		// ignore
	}
}

export async function clearWidgetBridge(): Promise<void> {
	if (!isNativeApp()) return;
	try {
		await WidgetBridge.clear();
	} catch {
		// ignore
	}
}
