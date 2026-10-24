import { expect, type Page } from '@playwright/test';

/** Redoc отрисовал спеку. Тег `<redoc>` есть и на экране «Something went wrong». */
export async function expectApiDocsRendered(page: Page) {
	await expect(page.getByRole('menuitem', { name: /Проверка состояния сервера/ })).toBeVisible({
		timeout: 20_000
	});
	await expect(page.getByText('Something went wrong')).toHaveCount(0);
}
