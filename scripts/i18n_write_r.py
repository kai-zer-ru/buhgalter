#!/usr/bin/env python3
"""Rewrite apperror.Write(..., message) to apperror.WriteR(w, r, ...) with locale keys."""

from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1] / "server" / "internal"

MSG_TO_KEY = {
    "некорректный JSON": "ERR_INVALID_JSON",
    "логин и пароль обязательны": "ERR_LOGIN_PASSWORD_REQUIRED",
    "логин должен быть от 3 до 32 символов": "ERR_LOGIN_LENGTH",
    "укажите имя администратора": "ERR_DISPLAY_NAME_REQUIRED",
    "имя не должно быть длиннее 64 символов": "ERR_DISPLAY_NAME_LENGTH",
    "пароли не совпадают": "PASSWORDS_MISMATCH",
    "external_url должен начинаться с http:// или https://": "ERR_EXTERNAL_URL",
    "для восстановления введите RESTORE": "ERR_RESTORE_CONFIRM",
    "файл не загружен": "ERR_FILE_REQUIRED",
    "ожидается файл .db": "ERR_DB_FILE_REQUIRED",
    "некорректное время бэкапа": "ERR_BACKUP_TIME",
    "notification_secret_key не должен быть пустым": "ERR_SECRET_KEY_EMPTY",
    "нельзя удалить себя": "ERR_CANNOT_DELETE_SELF",
    "язык должен быть ru или en": "ERR_LANGUAGE",
    "валюта должна быть RUB, USD или EUR": "ERR_CURRENCY",
    "тема должна быть light или dark": "ERR_THEME",
    "некорректный часовой пояс": "ERR_TIMEZONE",
    "channel должен быть telegram или max": "ERR_CHANNEL",
    "канал уведомлений не настроен": "ERR_NOTIFICATION_CHANNEL",
    "group_by: day, week или month": "ERR_GROUP_BY",
    "некорректный формат даты периода": "ERR_PERIOD_DATE",
    "некорректный multipart запрос": "ERR_MULTIPART",
    "не указан id задачи": "ERR_JOB_ID",
    "нельзя удалить отдельную ногу перевода": "ERR_TRANSFER_DELETE",
    "счёт с таким именем уже существует": "CONFLICT_ACCOUNT_NAME",
    "категория с таким именем уже существует": "CONFLICT_CATEGORY_NAME",
    "подкатегория с таким именем уже существует": "CONFLICT_SUBCATEGORY_NAME",
    "должник с таким именем уже существует": "CONFLICT_DEBTOR_NAME",
    "у должника есть активные долги": "CONFLICT_DEBTOR_ACTIVE_DEBTS",
    "долг уже закрыт": "CONFLICT_DEBT_CLOSED",
    "нельзя взять в долг у того, кому вы уже дали — закройте активный долг «дал в долг»": "CONFLICT_DEBT_CANNOT_BORROW",
    "нельзя дать в долг тому, у кого вы уже взяли — закройте активный долг «взял в долг»": "CONFLICT_DEBT_CANNOT_LEND",
    "логин уже занят": "CONFLICT_LOGIN_TAKEN",
    "требуется авторизация": "UNAUTHORIZED",
    "неверный логин или пароль": "INVALID_CREDENTIALS",
    "слишком много попыток входа": "RATE_LIMITED",
    "начальная настройка уже выполнена": "ALREADY_CONFIGURED",
    "сервер временно недоступен": "SERVICE_UNAVAILABLE",
    "внутренняя ошибка сервера": "INTERNAL_ERROR",
}

CODE_CONST = {
    '"INTERNAL_ERROR"': "apperror.InternalError",
    '"NOT_FOUND"': "apperror.NotFound",
    '"CONFLICT"': "apperror.Conflict",
    '"ALREADY_CONFIGURED"': "apperror.AlreadyConfigured",
    '"SERVICE_UNAVAILABLE"': "apperror.ServiceUnavailable",
    '"VALIDATION_ERROR"': "apperror.ValidationError",
}

WRITE_RE = re.compile(
    r'apperror\.Write\(w, (http\.Status[A-Za-z0-9]+), ([^,]+), ("(?:\\.|[^"\\])*"|apperror\.[A-Za-z]+)\)'
)


def rewrite(content: str) -> str:
    def repl(m: re.Match[str]) -> str:
        status, code, msg = m.group(1), m.group(2), m.group(3)
        code_out = CODE_CONST.get(code.strip(), code.strip())
        if msg.startswith("apperror."):
            return f"apperror.WriteR(w, r, {status}, {code_out})"
        raw = bytes(msg[1:-1], "utf-8").decode("unicode_escape") if "\\" in msg else msg[1:-1]
        key = MSG_TO_KEY.get(raw)
        if key:
            if key in {
                "UNAUTHORIZED",
                "INVALID_CREDENTIALS",
                "RATE_LIMITED",
                "PASSWORDS_MISMATCH",
                "ALREADY_CONFIGURED",
                "SERVICE_UNAVAILABLE",
                "INTERNAL_ERROR",
                "NOT_FOUND",
                "CONFLICT",
            } and code_out.startswith("apperror."):
                return f"apperror.WriteR(w, r, {status}, {code_out})"
            return f'apperror.WriteR(w, r, {status}, {code_out}, "{key}")'
        if code.strip() in ('"INTERNAL_ERROR"', "apperror.InternalError"):
            return f"apperror.WriteR(w, r, {status}, apperror.InternalError)"
        if code.strip() in ('"NOT_FOUND"', "apperror.NotFound"):
            return f"apperror.WriteR(w, r, {status}, apperror.NotFound)"
        if code.strip() in ('"CONFLICT"', "apperror.Conflict"):
            return f"apperror.WriteR(w, r, {status}, apperror.Conflict)"
        return m.group(0)

    return WRITE_RE.sub(repl, content)


def main() -> None:
    changed = 0
    for path in ROOT.rglob("*.go"):
        if path.name.endswith("_test.go"):
            continue
        text = path.read_text(encoding="utf-8")
        new = rewrite(text)
        if new != text:
            path.write_text(new, encoding="utf-8")
            changed += 1
            print(path.relative_to(ROOT.parent.parent))
    print(f"updated {changed} files")


if __name__ == "__main__":
    main()
