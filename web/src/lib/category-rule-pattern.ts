/** Default text rule pattern from transaction description (inbox «всегда так»). */
export function defaultPatternFromDescription(description: string): string {
	return description.trim().replace(/\s+/g, ' ').slice(0, 200);
}
