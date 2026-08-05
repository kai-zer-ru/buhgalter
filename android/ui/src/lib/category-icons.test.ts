import { describe, expect, it } from 'vitest';
import {
	defaultCategoryNameForIcon,
	getCategoryIconDef,
	iconMatchesKind,
	searchCategoryIcons
} from './category-icons';

/** Icons added in the v1.4.1 catalog expansion. */
const NEW_ICONS = [
	{ id: 'tram', kind: 'expense' as const, name: 'Трамвай', tag: 'трамвай' },
	{ id: 'trolleybus', kind: 'expense' as const, name: 'Троллейбус', tag: 'троллейбус' },
	{ id: 'motorcycle', kind: 'expense' as const, name: 'Мотоцикл', tag: 'мотоцикл' },
	{ id: 'car-wash', kind: 'expense' as const, name: 'Мойка авто', tag: 'мойка' },
	{ id: 'fine', kind: 'expense' as const, name: 'Штраф', tag: 'штраф' },
	{ id: 'laundry', kind: 'expense' as const, name: 'Стирка', tag: 'стирка' },
	{ id: 'hygiene', kind: 'expense' as const, name: 'Гигиена', tag: 'гигиена' },
	{ id: 'post', kind: 'expense' as const, name: 'Почта', tag: 'почта' },
	{ id: 'sauna', kind: 'expense' as const, name: 'Баня', tag: 'баня' },
	{ id: 'manicure', kind: 'expense' as const, name: 'Маникюр', tag: 'маникюр' },
	{ id: 'massage', kind: 'expense' as const, name: 'Массаж', tag: 'массаж' },
	{ id: 'glasses', kind: 'expense' as const, name: 'Очки', tag: 'очки' },
	{ id: 'psychologist', kind: 'expense' as const, name: 'Психолог', tag: 'психолог' },
	{ id: 'sweets', kind: 'expense' as const, name: 'Сладости', tag: 'сладости' },
	{ id: 'sushi', kind: 'expense' as const, name: 'Суши', tag: 'суши' },
	{ id: 'tea', kind: 'expense' as const, name: 'Чай', tag: 'чай' },
	{ id: 'fruit', kind: 'expense' as const, name: 'Фрукты', tag: 'фрукты' },
	{ id: 'meat', kind: 'expense' as const, name: 'Мясо', tag: 'мясо' },
	{ id: 'dairy', kind: 'expense' as const, name: 'Молочка', tag: 'молоко' },
	{ id: 'theater', kind: 'expense' as const, name: 'Театр', tag: 'театр' },
	{ id: 'museum', kind: 'expense' as const, name: 'Музей', tag: 'музей' },
	{ id: 'camping', kind: 'expense' as const, name: 'Кемпинг', tag: 'кемпинг' },
	{ id: 'fishing', kind: 'expense' as const, name: 'Рыбалка', tag: 'рыбалка' },
	{ id: 'vet', kind: 'expense' as const, name: 'Ветеринар', tag: 'ветеринар' },
	{ id: 'kindergarten', kind: 'expense' as const, name: 'Детский сад', tag: 'детский сад' },
	{ id: 'scholarship', kind: 'income' as const, name: 'Стипендия', tag: 'стипендия' },
	{ id: 'benefit', kind: 'income' as const, name: 'Пособие', tag: 'пособие' }
];

/** Brand icons for subscriptions (official logos). */
const SUBSCRIPTION_ICONS = [
	{ id: 'yandex-plus', name: 'Яндекс Плюс', tag: 'яндекс плюс' },
	{ id: 'yandex-music', name: 'Яндекс Музыка', tag: 'яндекс музыка' },
	{ id: 'kinopoisk', name: 'Кинопоиск', tag: 'кинопоиск' },
	{ id: 'spotify', name: 'Spotify', tag: 'spotify' },
	{ id: 'netflix', name: 'Netflix', tag: 'netflix' },
	{ id: 'youtube', name: 'YouTube', tag: 'youtube' },
	{ id: 'telegram', name: 'Telegram', tag: 'телеграм' },
	{ id: 'chatgpt', name: 'ChatGPT', tag: 'chatgpt' }
];

/** Brand icons for retail / loyalty. */
const STORE_ICONS = [
	{ id: 'gazprom-bonus', name: 'Газпром Бонус', tag: 'газпром бонус' },
	{ id: 'pyaterochka', name: 'Пятёрочка', tag: 'пятёрочка' },
	{ id: 'magnit', name: 'Магнит', tag: 'магнит' },
	{ id: 'perekrestok', name: 'Перекрёсток', tag: 'перекрёсток' },
	{ id: 'samokat', name: 'Самокат', tag: 'самокат' },
	{ id: 'mvideo', name: 'М.Видео', tag: 'мвидео' }
];

/** Bank brand icons (from data/banks/). */
const BANK_ICONS = [
	{ id: 'tinkoff', name: 'Т-Банк', tag: 'т-банк' },
	{ id: 'sberbank', name: 'Сбер', tag: 'сбербанк' },
	{ id: 'vtb', name: 'ВТБ', tag: 'втб' },
	{ id: 'alfabank', name: 'Альфа-Банк', tag: 'альфа' },
	{ id: 'yandex-bank', name: 'Яндекс Банк', tag: 'яндекс банк' }
];

describe('category icons catalog', () => {
	it.each(NEW_ICONS)('$id is in catalog with name and kind', ({ id, kind, name }) => {
		const def = getCategoryIconDef(id);
		expect(def).toBeDefined();
		expect(def?.name).toBe(name);
		expect(iconMatchesKind(id, kind)).toBe(true);
		expect(defaultCategoryNameForIcon(id, kind)).toBe(name);
	});

	it.each(NEW_ICONS)('$id is findable by tag "$tag"', ({ id, kind, tag }) => {
		const found = searchCategoryIcons(tag, kind);
		expect(found.some((icon) => icon.id === id)).toBe(true);
	});

	it.each(SUBSCRIPTION_ICONS)('$id subscription brand is in catalog', ({ id, name }) => {
		const def = getCategoryIconDef(id);
		expect(def).toBeDefined();
		expect(def?.name).toBe(name);
		expect(iconMatchesKind(id, 'expense')).toBe(true);
		expect(def?.official_logo).toBe(true);
	});

	it.each(SUBSCRIPTION_ICONS)('$id is findable by tag "$tag"', ({ id, tag }) => {
		const found = searchCategoryIcons(tag, 'expense');
		expect(found.some((icon) => icon.id === id)).toBe(true);
	});

	it('search matches icon name', () => {
		const found = searchCategoryIcons('яндекс плюс', 'expense');
		expect(found.some((icon) => icon.id === 'yandex-plus')).toBe(true);
	});

	it.each(STORE_ICONS)('$id store brand is in catalog', ({ id, name }) => {
		const def = getCategoryIconDef(id);
		expect(def).toBeDefined();
		expect(def?.name).toBe(name);
		expect(iconMatchesKind(id, 'expense')).toBe(true);
		expect(def?.official_logo).toBe(true);
	});

	it.each(STORE_ICONS)('$id is findable by tag "$tag"', ({ id, tag }) => {
		const found = searchCategoryIcons(tag, 'expense');
		expect(found.some((icon) => icon.id === id)).toBe(true);
	});

	it.each(BANK_ICONS)('$id bank brand is in catalog', ({ id, name }) => {
		const def = getCategoryIconDef(id);
		expect(def).toBeDefined();
		expect(def?.name).toBe(name);
		expect(iconMatchesKind(id, 'expense')).toBe(true);
		expect(def?.official_logo).toBe(true);
	});

	it.each(BANK_ICONS)('$id is findable by tag "$tag"', ({ id, tag }) => {
		const found = searchCategoryIcons(tag, 'expense');
		expect(found.some((icon) => icon.id === id)).toBe(true);
	});
});
