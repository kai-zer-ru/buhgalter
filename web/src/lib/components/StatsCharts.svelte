<script lang="ts">
	import { onDestroy, onMount, tick } from 'svelte';
	import { _, locale } from 'svelte-i18n';
	import { tr } from '$lib/i18n';
	import Chart from 'chart.js/auto';
	import type { ChartConfiguration } from 'chart.js/auto';
	import type { StatsCategoryItem, StatsNetWorthSnapshot, StatsPeriodItem } from '$lib/api/client';

	let {
		netWorthItems,
		byPeriod,
		expenseCategories,
		periodLabel
	}: {
		netWorthItems: StatsNetWorthSnapshot[];
		byPeriod: StatsPeriodItem[];
		expenseCategories: StatsCategoryItem[];
		periodLabel: (period: string) => string;
	} = $props();

	let netCanvas = $state<HTMLCanvasElement | null>(null);
	let periodCanvas = $state<HTMLCanvasElement | null>(null);
	let categoryCanvas = $state<HTMLCanvasElement | null>(null);

	let chartsReady = $state(false);
	const charts: Chart[] = [];

	const netChron = $derived([...netWorthItems].reverse());
	const periodChron = $derived([...byPeriod].reverse());
	const topExpense = $derived(expenseCategories.slice(0, 8));
	const showNetChart = $derived(netChron.length > 0);
	const showPeriodChart = $derived(periodChron.length > 0);
	const showCategoryChart = $derived(topExpense.length > 0);

	function cssVar(name: string, fallback: string): string {
		if (typeof document === 'undefined') return fallback;
		const v = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
		return v || fallback;
	}

	function destroyCharts() {
		for (const c of charts) c.destroy();
		charts.length = 0;
	}

	function buildCharts(localeHint: string | undefined = undefined) {
		if (!chartsReady) return;
		destroyCharts();
		void (localeHint ?? $locale);
		const text = cssVar('--text', '#0f172a');
		Chart.defaults.color = text;
		const muted = cssVar('--text-muted', '#475569');
		const border = cssVar('--border', '#e2e8f0');
		const accent = cssVar('--primary', '#047857');
		const expenseColor = cssVar('--danger', '#dc2626');
		const incomeColor = cssVar('--primary', '#059669');

		if (netCanvas && netChron.length > 0) {
			const cfg: ChartConfiguration<'line'> = {
				type: 'line',
				data: {
					labels: netChron.map((r) => periodLabel(r.period ?? '')),
					datasets: [
						{
							label: tr('stats.netWorth.title'),
							data: netChron.map((r) => r.net_worth / 100),
							borderColor: accent,
							backgroundColor: accent,
							tension: 0.25,
							fill: false
						}
					]
				},
				options: {
					responsive: true,
					maintainAspectRatio: false,
					plugins: { legend: { labels: { color: text } } },
					scales: {
						x: { ticks: { color: muted }, grid: { color: border } },
						y: { ticks: { color: muted }, grid: { color: border } }
					}
				}
			};
			charts.push(new Chart(netCanvas, cfg));
		}

		if (periodCanvas && periodChron.length > 0) {
			const cfg: ChartConfiguration<'bar'> = {
				type: 'bar',
				data: {
					labels: periodChron.map((r) => periodLabel(r.period)),
					datasets: [
						{
							label: tr('stats.summary.income'),
							data: periodChron.map((r) => r.income / 100),
							backgroundColor: incomeColor
						},
						{
							label: tr('stats.summary.expense'),
							data: periodChron.map((r) => r.expense / 100),
							backgroundColor: expenseColor
						}
					]
				},
				options: {
					responsive: true,
					maintainAspectRatio: false,
					plugins: { legend: { labels: { color: text } } },
					scales: {
						x: { ticks: { color: muted }, grid: { color: border } },
						y: { ticks: { color: muted }, grid: { color: border } }
					}
				}
			};
			charts.push(new Chart(periodCanvas, cfg));
		}

		if (categoryCanvas && topExpense.length > 0) {
			const palette = [
				accent,
				expenseColor,
				incomeColor,
				'#9333ea',
				'#ea580c',
				'#0891b2',
				'#ca8a04',
				'#64748b'
			];
			const cfg: ChartConfiguration<'doughnut'> = {
				type: 'doughnut',
				data: {
					labels: topExpense.map((r) => r.category_name),
					datasets: [
						{
							data: topExpense.map((r) => r.total / 100),
							backgroundColor: palette
						}
					]
				},
				options: {
					responsive: true,
					maintainAspectRatio: false,
					plugins: { legend: { position: 'bottom', labels: { color: text } } }
				}
			};
			charts.push(new Chart(categoryCanvas, cfg));
		}
	}

	onMount(async () => {
		chartsReady = true;
		await tick();
		buildCharts();
	});

	$effect(() => {
		if (!chartsReady) return;
		const hasData = netWorthItems.length > 0 || byPeriod.length > 0 || expenseCategories.length > 0;
		if (!hasData) {
			destroyCharts();
			return;
		}
		const localeTag = String($locale ?? '');
		void tick().then(() => buildCharts(localeTag));
	});

	onDestroy(() => destroyCharts());
</script>

{#if showNetChart || showPeriodChart || showCategoryChart}
	<div class="grid gap-4 lg:grid-cols-2">
		{#if showNetChart}
			<div class="card">
				<h2 class="mb-2 text-lg font-medium">{$_('stats.charts.netWorth')}</h2>
				<div class="relative h-56">
					<canvas bind:this={netCanvas} aria-label={$_('stats.charts.netWorth')}></canvas>
				</div>
			</div>
		{/if}
		{#if showPeriodChart}
			<div class="card">
				<h2 class="mb-2 text-lg font-medium">{$_('stats.charts.byPeriod')}</h2>
				<div class="relative h-56">
					<canvas bind:this={periodCanvas} aria-label={$_('stats.charts.byPeriod')}></canvas>
				</div>
			</div>
		{/if}
		{#if showCategoryChart}
			<div class="card lg:col-span-2">
				<h2 class="mb-2 text-lg font-medium">{$_('stats.charts.expensesByCategory')}</h2>
				<div class="relative mx-auto h-72 max-w-md">
					<canvas bind:this={categoryCanvas} aria-label={$_('stats.charts.expensesByCategory')}
					></canvas>
				</div>
			</div>
		{/if}
	</div>
{/if}
