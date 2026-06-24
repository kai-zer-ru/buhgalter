#!/usr/bin/env python3
"""Regenerate data/category_icons.json with kind scopes and marketplace brands."""

from __future__ import annotations

import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "data" / "category_icons.json"

# kind: expense | income | both
ICONS: list[dict] = [
    {"id": "transport", "emoji": "🚌", "kind": "expense", "tags": ["транспорт", "автобус"]},
    {"id": "food", "emoji": "🛒", "kind": "expense", "tags": ["магазин", "еда", "покупки"]},
    {"id": "groceries", "emoji": "🥫", "kind": "expense", "tags": ["продукты", "бакалея", "еда"]},
    {"id": "phone", "emoji": "📱", "kind": "expense", "tags": ["связь", "телефон", "мобильный"]},
    {"id": "health", "emoji": "❤️", "kind": "expense", "tags": ["здоровье", "медицина"]},
    {"id": "salary", "emoji": "💰", "kind": "income", "tags": ["зарплата", "доход", "деньги"]},
    {"id": "home", "emoji": "🏠", "kind": "expense", "tags": ["дом", "жильё", "квартира"]},
    {"id": "default", "emoji": "📁", "kind": "both", "tags": ["разное", "прочее"]},
    {"id": "wildberries", "official_logo": True, "kind": "expense", "tags": ["wildberries", "вб", "маркетплейс", "шопинг"]},
    {"id": "ozon", "official_logo": True, "kind": "expense", "tags": ["ozon", "озон", "маркетплейс"]},
    {"id": "yandex-market", "official_logo": True, "kind": "expense", "tags": ["яндекс маркет", "маркетплейс", "яндекс"]},
    {"id": "avito", "official_logo": True, "kind": "income", "tags": ["авито", "продажа", "барахолка", "доход"]},
    {"id": "car", "emoji": "🚗", "kind": "expense", "tags": ["авто", "машина", "транспорт"]},
    {"id": "taxi", "emoji": "🚕", "kind": "expense", "tags": ["такси", "транспорт"]},
    {"id": "train", "emoji": "🚆", "kind": "expense", "tags": ["поезд", "жд", "транспорт"]},
    {"id": "plane", "emoji": "✈️", "kind": "expense", "tags": ["самолёт", "авиа", "перелёт"]},
    {"id": "metro", "emoji": "🚇", "kind": "expense", "tags": ["метро", "транспорт"]},
    {"id": "bike", "emoji": "🚲", "kind": "expense", "tags": ["велосипед", "транспорт"]},
    {"id": "fuel", "emoji": "⛽", "kind": "expense", "tags": ["бензин", "топливо", "азс"]},
    {"id": "parking", "emoji": "🅿️", "kind": "expense", "tags": ["парковка", "стоянка"]},
    {"id": "ship", "emoji": "🚢", "kind": "expense", "tags": ["корабль", "паром", "море"]},
    {"id": "scooter", "emoji": "🛴", "kind": "expense", "tags": ["самокат", "транспорт"]},
    {"id": "restaurant", "emoji": "🍽️", "kind": "expense", "tags": ["ресторан", "кафе", "еда"]},
    {"id": "coffee", "emoji": "☕", "kind": "expense", "tags": ["кофе", "кафе"]},
    {"id": "fastfood", "emoji": "🍔", "kind": "expense", "tags": ["фастфуд", "бургер", "еда"]},
    {"id": "pizza", "emoji": "🍕", "kind": "expense", "tags": ["пицца", "еда"]},
    {"id": "grocery", "emoji": "🥬", "kind": "expense", "tags": ["овощи", "продукты"]},
    {"id": "bakery", "emoji": "🥖", "kind": "expense", "tags": ["хлеб", "выпечка"]},
    {"id": "beer", "emoji": "🍺", "kind": "expense", "tags": ["алкоголь", "пиво", "бар"]},
    {"id": "wine", "emoji": "🍷", "kind": "expense", "tags": ["вино", "алкоголь"]},
    {"id": "water", "emoji": "💧", "kind": "expense", "tags": ["вода", "коммуналка"]},
    {"id": "electricity", "emoji": "⚡", "kind": "expense", "tags": ["электричество", "свет", "коммуналка"]},
    {"id": "gas", "emoji": "🔥", "kind": "expense", "tags": ["газ", "отопление", "коммуналка"]},
    {"id": "utilities", "emoji": "🧾", "kind": "expense", "tags": ["коммуналка", "квитанции", "жкх"]},
    {"id": "internet", "emoji": "🌐", "kind": "expense", "tags": ["интернет", "связь"]},
    {"id": "wifi", "emoji": "📶", "kind": "expense", "tags": ["wifi", "связь"]},
    {"id": "tv", "emoji": "📺", "kind": "expense", "tags": ["телевизор", "подписка", "стриминг"]},
    {"id": "subscription", "emoji": "🔁", "kind": "expense", "tags": ["подписка", "сервис"]},
    {"id": "pharmacy", "emoji": "💊", "kind": "expense", "tags": ["аптека", "лекарства", "здоровье"]},
    {"id": "hospital", "emoji": "🏥", "kind": "expense", "tags": ["больница", "клиника"]},
    {"id": "dentist", "emoji": "🦷", "kind": "expense", "tags": ["стоматолог", "зубы"]},
    {"id": "fitness", "emoji": "💪", "kind": "expense", "tags": ["спортзал", "фитнес"]},
    {"id": "yoga", "emoji": "🧘", "kind": "expense", "tags": ["йога", "спорт"]},
    {"id": "rent", "emoji": "🔑", "kind": "expense", "tags": ["аренда", "квартира", "жильё"]},
    {"id": "furniture", "emoji": "🛋️", "kind": "expense", "tags": ["мебель", "дом"]},
    {"id": "repair", "emoji": "🔧", "kind": "expense", "tags": ["ремонт", "инструменты"]},
    {"id": "cleaning", "emoji": "🧹", "kind": "expense", "tags": ["уборка", "клининг"]},
    {"id": "garden", "emoji": "🌱", "kind": "expense", "tags": ["сад", "растения", "дача"]},
    {"id": "lamp", "emoji": "💡", "kind": "expense", "tags": ["освещение", "лампа"]},
    {"id": "clothes", "emoji": "👕", "kind": "expense", "tags": ["одежда", "шопинг"]},
    {"id": "shoes", "emoji": "👟", "kind": "expense", "tags": ["обувь"]},
    {"id": "bag", "emoji": "👜", "kind": "expense", "tags": ["сумка", "аксессуары"]},
    {"id": "jewelry", "emoji": "💍", "kind": "expense", "tags": ["украшения", "бижутерия"]},
    {"id": "beauty", "emoji": "💄", "kind": "expense", "tags": ["косметика", "красота"]},
    {"id": "haircut", "emoji": "💇", "kind": "expense", "tags": ["парикмахер", "стрижка"]},
    {"id": "entertainment", "emoji": "🎬", "kind": "expense", "tags": ["развлечения", "кино"]},
    {"id": "movie", "emoji": "🍿", "kind": "expense", "tags": ["кино", "фильм"]},
    {"id": "game", "emoji": "🎮", "kind": "expense", "tags": ["игры", "приставка"]},
    {"id": "music", "emoji": "🎵", "kind": "expense", "tags": ["музыка", "концерт"]},
    {"id": "ticket", "emoji": "🎫", "kind": "expense", "tags": ["билеты", "мероприятие"]},
    {"id": "sport", "emoji": "⚽", "kind": "expense", "tags": ["спорт", "футбол"]},
    {"id": "tennis", "emoji": "🎾", "kind": "expense", "tags": ["теннис", "спорт"]},
    {"id": "ski", "emoji": "⛷️", "kind": "expense", "tags": ["лыжи", "зима", "спорт"]},
    {"id": "swim", "emoji": "🏊", "kind": "expense", "tags": ["бассейн", "плавание"]},
    {"id": "education", "emoji": "📚", "kind": "expense", "tags": ["образование", "книги"]},
    {"id": "book", "emoji": "📖", "kind": "expense", "tags": ["книга", "чтение"]},
    {"id": "school", "emoji": "🏫", "kind": "expense", "tags": ["школа", "обучение"]},
    {"id": "university", "emoji": "🎓", "kind": "expense", "tags": ["университет", "вуз"]},
    {"id": "course", "emoji": "📝", "kind": "expense", "tags": ["курсы", "обучение"]},
    {"id": "child", "emoji": "🧒", "kind": "expense", "tags": ["ребёнок", "дети"]},
    {"id": "baby", "emoji": "👶", "kind": "expense", "tags": ["малыш", "дети"]},
    {"id": "toy", "emoji": "🧸", "kind": "expense", "tags": ["игрушки", "дети"]},
    {"id": "pet", "emoji": "🐾", "kind": "expense", "tags": ["питомец", "животные"]},
    {"id": "dog", "emoji": "🐕", "kind": "expense", "tags": ["собака"]},
    {"id": "cat", "emoji": "🐈", "kind": "expense", "tags": ["кошка"]},
    {"id": "gift", "emoji": "🎁", "kind": "expense", "tags": ["подарок", "подарки"]},
    {"id": "charity", "emoji": "🤝", "kind": "expense", "tags": ["благотворительность", "помощь"]},
    {"id": "tax", "emoji": "🧮", "kind": "expense", "tags": ["налоги", "налог"]},
    {"id": "insurance", "emoji": "🛡️", "kind": "expense", "tags": ["страховка"]},
    {"id": "bank-fee", "emoji": "🏦", "kind": "expense", "tags": ["банк", "комиссия"]},
    {"id": "loan", "emoji": "📉", "kind": "expense", "tags": ["кредит", "долг", "платёж"]},
    {"id": "legal", "emoji": "⚖️", "kind": "expense", "tags": ["юрист", "суд", "право"]},
    {"id": "office", "emoji": "💼", "kind": "expense", "tags": ["офис", "работа", "канцелярия"]},
    {"id": "software", "emoji": "💻", "kind": "expense", "tags": ["софт", "программы", "лицензия"]},
    {"id": "cloud", "emoji": "☁️", "kind": "expense", "tags": ["облако", "хостинг"]},
    {"id": "delivery", "emoji": "📦", "kind": "expense", "tags": ["доставка", "посылка"]},
    {"id": "travel", "emoji": "🧳", "kind": "expense", "tags": ["путешествие", "отпуск"]},
    {"id": "hotel", "emoji": "🏨", "kind": "expense", "tags": ["отель", "проживание"]},
    {"id": "beach", "emoji": "🏖️", "kind": "expense", "tags": ["пляж", "отдых"]},
    {"id": "mountain", "emoji": "🏔️", "kind": "expense", "tags": ["горы", "поход"]},
    {"id": "camera", "emoji": "📷", "kind": "expense", "tags": ["фото", "техника"]},
    {"id": "electronics", "emoji": "🔌", "kind": "expense", "tags": ["электроника", "техника"]},
    {"id": "phone-device", "emoji": "📲", "kind": "expense", "tags": ["смартфон", "гаджет"]},
    {"id": "laptop", "emoji": "💻", "kind": "expense", "tags": ["ноутбук", "компьютер"]},
    {"id": "wedding", "emoji": "💒", "kind": "expense", "tags": ["свадьба", "торжество"]},
    {"id": "party", "emoji": "🎉", "kind": "expense", "tags": ["праздник", "вечеринка"]},
    {"id": "flowers", "emoji": "💐", "kind": "expense", "tags": ["цветы"]},
    {"id": "smoke", "emoji": "🚬", "kind": "expense", "tags": ["сигареты", "табак"]},
    {"id": "income-other", "emoji": "💵", "kind": "income", "tags": ["доход", "прочее"]},
    {"id": "wallet", "emoji": "👛", "kind": "income", "tags": ["кошелёк", "доход"]},
    {"id": "cash", "emoji": "💴", "kind": "income", "tags": ["наличные", "доход"]},
    {"id": "bonus", "emoji": "🎯", "kind": "income", "tags": ["премия", "бонус", "доход"]},
    {"id": "freelance", "emoji": "🧑‍💻", "kind": "income", "tags": ["фриланс", "подработка"]},
    {"id": "rental-income", "emoji": "🏘️", "kind": "income", "tags": ["аренда", "доход", "сдача"]},
    {"id": "refund", "emoji": "↩️", "kind": "income", "tags": ["возврат", "кэшбэк"]},
    {"id": "cashback", "emoji": "🔙", "kind": "income", "tags": ["кэшбэк", "возврат"]},
    {"id": "dividend", "emoji": "📈", "kind": "income", "tags": ["дивиденды", "инвестиции"]},
    {"id": "investment", "emoji": "💹", "kind": "income", "tags": ["инвестиции", "биржа"]},
    {"id": "pension", "emoji": "👴", "kind": "income", "tags": ["пенсия", "доход"]},
    {"id": "sale", "emoji": "🏷️", "kind": "income", "tags": ["продажа", "доход"]},
    {"id": "gift-income", "emoji": "🎀", "kind": "income", "tags": ["подарок", "получено", "доход"]},
    {"id": "lottery", "emoji": "🎰", "kind": "income", "tags": ["лотерея", "выигрыш"]},
    {"id": "tips", "emoji": "🪙", "kind": "income", "tags": ["чаевые", "доход"]},
    {"id": "grant", "emoji": "🏛️", "kind": "income", "tags": ["грант", "субсидия"]},
    {"id": "alimony", "emoji": "👨‍👩‍👧", "kind": "income", "tags": ["алименты", "доход"]},
    {"id": "interest", "emoji": "🏧", "kind": "income", "tags": ["проценты", "вклад"]},
    {"id": "crypto", "emoji": "₿", "kind": "income", "tags": ["крипто", "биткоин"]},
    {"id": "side-job", "emoji": "🛠️", "kind": "income", "tags": ["подработка", "халтура"]},
    {"id": "royalty", "emoji": "©️", "kind": "income", "tags": ["роялти", "авторские"]},
    {"id": "compensation", "emoji": "🧾", "kind": "both", "tags": ["компенсация", "возмещение"]},
]

# Имена, если автоматическое из tags[0] не подходит.
NAME_OVERRIDES: dict[str, str] = {
    "food": "Магазины",
    "income-other": "Прочие доходы",
    "wildberries": "Wildberries",
    "ozon": "Ozon",
    "yandex-market": "Яндекс Маркет",
    "avito": "Авито",
    "gift-income": "Подарок",
    "fastfood": "Фастфуд",
    "wifi": "Wi‑Fi",
    "crypto": "Крипто",
}


def title_name(text: str) -> str:
    return " ".join(part[:1].upper() + part[1:] if part else part for part in text.split())


def finalize_icon(icon: dict) -> dict:
    out = dict(icon)
    icon_id = out["id"]
    if "name" not in out:
        out["name"] = NAME_OVERRIDES.get(icon_id, title_name(out["tags"][0]))
    return out

QUICK = {
    "expense": ["transport", "groceries", "food", "wildberries", "ozon", "yandex-market", "default"],
    "income": ["salary", "sale", "avito", "freelance", "rental-income", "bonus", "default"],
}

def main() -> int:
    ids = {i["id"] for i in ICONS}
    if len(ids) != len(ICONS):
        raise SystemExit("duplicate icon ids")

    for kind, quick_ids in QUICK.items():
        for qid in quick_ids:
            if qid not in ids:
                raise SystemExit(f"quick {kind}: unknown icon {qid}")
            icon = next(i for i in ICONS if i["id"] == qid)
            if icon["kind"] not in (kind, "both"):
                raise SystemExit(f"quick {kind}: {qid} is {icon['kind']}")

    data = {"quick": QUICK, "icons": [finalize_icon(i) for i in ICONS]}
    OUT.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"OK {len(ICONS)} icons -> {OUT}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
