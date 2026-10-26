import { readdirSync, readFileSync } from 'node:fs';
import { join, relative } from 'node:path';
import { describe, expect, it } from 'vitest';

const SRC = join(import.meta.dirname, '..');

function svelteFiles(dir: string): string[] {
	return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
		const path = join(dir, entry.name);
		if (entry.isDirectory()) return svelteFiles(path);
		return entry.name.endsWith('.svelte') ? [path] : [];
	});
}

/** Opening `<input …>` tags (attributes may span lines; `>` inside `{…}` is skipped). */
function inputTags(source: string): string[] {
	const tags: string[] = [];
	let from = 0;
	for (;;) {
		const start = source.indexOf('<input', from);
		if (start < 0) return tags;
		let depth = 0;
		let end = start;
		for (; end < source.length; end++) {
			const ch = source[end];
			if (ch === '{') depth++;
			else if (ch === '}') depth--;
			else if (ch === '>' && depth === 0) break;
		}
		tags.push(source.slice(start, end + 1));
		from = end + 1;
	}
}

describe('android UI invariants', () => {
	it('every <input type="time"> uses nativeTimePicker', () => {
		const missing: string[] = [];
		let count = 0;
		for (const file of svelteFiles(SRC)) {
			for (const tag of inputTags(readFileSync(file, 'utf8'))) {
				if (!/\btype="time"/.test(tag)) continue;
				count++;
				if (!tag.includes('use:nativeTimePicker')) missing.push(relative(SRC, file));
			}
		}
		expect(count).toBeGreaterThan(0);
		expect(missing).toEqual([]);
	});

	it('ModalShell portals the overlay to document.body', () => {
		const source = readFileSync(join(SRC, 'lib/components/ModalShell.svelte'), 'utf8');
		expect(source).toContain('use:portal');
		expect(source).toContain('document.body');
	});
});
