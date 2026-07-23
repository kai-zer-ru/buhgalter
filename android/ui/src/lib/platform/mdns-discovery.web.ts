export class LanDiscoveryWeb {
	async discover(): Promise<{ servers: { host: string; port: number }[] }> {
		return { servers: [] };
	}
}
