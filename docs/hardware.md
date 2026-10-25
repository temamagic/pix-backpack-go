# Historical hardware observations

These results were recorded in the supplied project notes on 2026-10-24. They
were not repeated during release preparation. Verified below means historically
reported visual/physical evidence, not a new test by this audit. Extracted-font
text tests describe the prior implementation; the public release uses caller fonts.

# Проверка на рюкзаке — 24 октября 2026

Устройство: PIX Backpack (identifier redacted), service Original V1 `00000100-e984-11e7-b78e-ffd6fcc34510`. Firmware 2.5.1, hardware 3.2, экран 16×20. macOS ARM64 / CoreBluetooth, сопряжение выполнено. Яркость на старте 76/255 (~29%), не менялась тестами.

| Проверка | Результат |
|---|---|
| PNG orientation, 326-byte RPC frame | пользователь подтвердил картинку |
| Live: диапазоны пикселей без clear/reset | пользователь подтвердил движущуюся оранжевую точку |
| Clock, Europe/Moscow | пользователь сообщил, что время было правильным |
| Countdown, Stopwatch, Bike, Crawler, Blocks, Breaker | все записи подтверждены GATT; пользователь видел переключение нескольких встроенных режимов |
| Scrolling text `PIX from Go` | пользователь подтвердил бегущую строку |
| Aurora, 200 цветов / 8 кадров | вся загрузка завершилась; проверена передача двух частей палитры |
| Matrix Rain, 60 кадров | загрузка в память завершилась; в конце сессии оставлена активной, BLE отключён |
| RPC notifications | подписка установилась, сообщений в ходе теста не получено |

Записи GATT подтверждают доставку значения, а не каждый отдельный визуальный эффект. Не заявляется, что пользователь проверил все игровые кнопки или каждую из 25 анимаций глазами.

## Измерения потока

В начале Stream делает setup: режим NONE, число кадров 0, палитра, кадр 0, длительность, направление, число кадров 1. Ни один живой кадр после setup не очищает экран, не сбрасывает число кадров и не пишет flash.

| Тест | Запрошено | Обработано | Elapsed без setup | Фактический темп | RPC bytes без setup |
|---|---:|---:|---:|---:|---:|
| точка, первый тест | 10 fps | 73 | 7.521 s | 9.71 fps | 576 |
| точка, повтор с визуальной проверкой | 10 fps | 273 | 29.195 s | 9.35 fps | 2176 |
| Rainbow Wave, целые кадры | 10 fps | 54 | 5.470 s | 9.87 fps | 17278 |
| Rainbow Wave, целые кадры | 30 fps | 104 | 7.590 s | 13.70 fps | 33578 |

Это конкретные замеры при RSSI примерно от -82 до -58 dBm, без гарантии максимума для других условий. Они включают ожидание завершения write-with-response и задержку планировщика. RPC bytes не включают ATT/L2CAP/radio overhead. Для обновления одного изменённого пикселя достаточно 7 байт RPC; двух соседних — 8 байт. Полный кадр — 326 байт.

Поток не является загрузкой видеофайла в рюкзак: `SET_FRAME` меняет содержимое активного кадра, firmware продолжает выводить его. Если процесс остановить, остаётся последний кадр. Для плавного автономного показа коротких циклов выгоднее один раз загрузить всю анимацию вместе с длительностями.

## Ёмкость 2048

Характеристика `0105` вернула `00 08`, то есть uint16le 2048. И Android 1.40, и iOS `_setFrameCount` передают count одним элементом массива, который превращается в один байт. Поэтому нельзя заключить, что для показа 2048 кадров достаточно послать uint16: такое изменение не подтверждено кодом или устройством. В библиотеке явно оставлены границы восстановленного формата. Stream обходит этот вопрос, используя только frame 0.

## Что не проверено на железе

Сохранение после перезапуска питания, формат RPC-ответов, все варианты watch face/alarm, Mini/legacy firmware, несколько устройств одновременно, Linux/Windows, пределы memory/config buffers вне форматов приложений. OTA и смена PIN не выполнялись.

Офлайн: все 25 анимаций / 259 кадров прошли проверку ограничений; отдельные экспортированные pixpack дают идентичные RPC-пакеты исходным элементам коллекции. Raw collection reports and original artwork remain in the private archive outside this repository.


# Bilingual font hardware verification — 2026-10-24

New tests during the font implementation, separate from the historical results
above. Platform: macOS Apple Silicon / CoreBluetooth. PIX Original V1, firmware
2.5.1, hardware 3.2, 16×20. No device identifiers or raw personal logs are included.
The native CLI embedded the macOS Bluetooth privacy metadata.

| Test | Delivery result | Human visual confirmation | Confidence |
|---|---|---|---|
| Built-in `mono`, green, 80 ms; `Hello, PIX! Привет, мир! Ёжик 123` | All writes completed without error | Both languages readable, Ё visible, text scrolling | Verified |
| External JSON through `LoadFont` / `-font-file`, based on `mono-bold`, with an original custom ★ glyph; orange, 90 ms; `Bold ★ Hello! Привет! Ёё 123` | All writes completed without error | Both languages readable, visibly bold letters and custom symbol visible | Verified |

The owner explicitly confirmed both tests while watching the physical backpack.
The external JSON file was generated outside the repository for this experiment;
OFL-derived built-in bitmaps and the reproducible conversion are distributed with
font licensing notices. The second test exercises the general user-font file
loader, not the built-in lookup path. `mono-bold` uses the same bundled glyph data
as that test's base, with the test-only custom symbol added separately.

Both tests set scrolling mode and loaded glyphs/config into RAM. No Save, Restart,
OTA, PIN, brightness or demo-setting commands were issued. BLE was closed after
each upload; the last orange scrolling message was left active. No claims are
made about power-cycle persistence, Mini, other firmware or other OS transports.
