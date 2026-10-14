# UX polish — скроллбары, pull-to-refresh, темы

**v1.8.1** ([ROADMAP](../ROADMAP.md#v181)). Спека: `bmad_docs/specs/spec-ux-polish/`.

**Статус (2026-10-14):** реализация закрыта (stories 1–4 done). Идёт **ручная приёмка ~неделю**. Найденные баги — **отдельными коммитами** в 1.8.1, не раздувать scope.

Стек: SvelteKit + Tailwind + свои компоненты (web + `android/ui`). Не SMUI / полный Material kit.

## Контекст

Боль: полосы прокрутки (особенно Android WebView), нет PTR текущего экрана, «топорность», плюс выбор look — **классика** или **MaterialBased**.

## Решения (зафиксировано 2026-10-14)

| Вопрос | Решение |
|--------|---------|
| Глубина MaterialBased | Токены + материализация Button / inputs / Select·Combobox / cards |
| Хранение classic/material | `localStorage` (`ui_style`), web и Android раздельно; не сервер |
| Порядок работ | Один пакет: скроллбары + PTR + polish + темы |
| PTR на web | Capacitor + web mobile viewport / PWA; не desktop |
| Radius MaterialBased | Единая шкала `0.75rem` (control / button / card) |
| Профиль | «Стиль интерфейса» в одной card с формой; без hint; сразу localStorage |
| Подпись light/dark | «Цветовая схема» / «Color scheme» |

## Scope (сделано)

1. **Скроллбары** — hide на touch/Android (+ `native-app` gutter fix); desktop thin/overlay.
2. **PTR** — soft revalidate текущего маршрута; Capacitor + web mobile; без Sync All / outbox на background.
3. **classic / MaterialBased** — `ui_style`, токены unlayered, материализация контролов.
4. **Polish follow-up** — radius unify; merge ui_style в profile card. Плотность списков / CTA / EmptyState — **отложены** (не блокер закрытия).

## Не входит

- SMUI / Material Web / big-bang kit.
- Синхронизация `ui_style` через API.
- Полный sync / heavy warm по PTR.
- Продуктовые фичи 1.9.0.

## Критерии / soak

- [x] Android/touch: нет видимых вертикальных scrollbar на основных экранах.
- [x] PTR на ключевых списках → soft reload, без Sync All; web mobile + Capacitor.
- [x] Переключатель classic / MaterialBased; `localStorage`.
- [x] MaterialBased: единый radius; токены не проигрывают cascade.
- [x] Профиль: ui_style в одной card с формой.
- [x] Vitest `ui_style` / PTR helpers; web e2e смена стиля.
- [ ] Soak: web desktop не ухудшен без причины.
- [ ] Soak: Android ручная приёмка classic ↔ MaterialBased (устройство).

Баги из soak → отдельные коммиты + checkbox’ы здесь/ROADMAP по факту.
