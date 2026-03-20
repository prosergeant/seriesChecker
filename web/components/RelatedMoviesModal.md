# RelatedMoviesModal.tsx

Модальное окно с двумя вкладками: "Похожие" (`/api/series/{id}/similar`) и "Связанные" (`/api/series/{id}/relations`).

## Поведение
- Открывается кнопкой `MoreHorizontal` внутри `ProgressCard`
- Данные загружаются лениво (только при открытии нужной вкладки)
- После нажатия "Отслеживать" — модалка **не закрывается**, чтобы можно было добавить несколько элементов за раз
- При добавлении вызывает `api.progress.update` со статусом `"planned"` и инвалидирует `["progress"]`

## Важные детали
- `SimilarMovie` использует `filmId`, `RelationMovie` — `kinopoiskId` (разные поля id, обрабатывается через `"filmId" in movie`)
- Типы отношений: SEQUEL, PREQUEL, SPIN_OFF, SPUN_OFF_FROM, REFERENCES_IN, EDITED_INTO, SIMILAR
- Модалка построена на `@base-ui/react/dialog` (не Radix UI)

## Почему модалка закрывалась (баг, исправлен в page.tsx)
`ProgressCard` использовал ключ с `index`: `progress-${series_id}-${index}`. При добавлении нового элемента список перерисовывался, индексы сдвигались, карточки ремаунтились — стейт `isOpen` сбрасывался. Фикс: ключ изменён на `item.id`.
