# PIX BLE protocol

These independently written notes retain protocol knowledge without app source.
Evidence labels:

- **Verified:** visual/device results reported in prior project notes; not rerun.
- **Observed:** recovered app constants, byte formats or successful GATT writes.
- **Inferred:** consequences of known formats and library safety limits.
- **Unknown:** not established; do not rely on it.

| Element | Confidence |
|---|---|
| Original V1 UUID, size 16×20, firmware 2.5.1 | Verified (historical) |
| Basic RAM upload, live partial offsets, clock, scrolling concept | Verified (historical) |
| RGB order, palette split, call/input counters, widget encodings | Observed; covered by synthetic tests |
| Games and bike/stopwatch complete semantics | Observed; visual coverage incomplete |
| Legacy/Mini UUIDs, OTA characteristic names, demo/brightness encoding | Observed in apps; hardware coverage incomplete |
| 512-byte GATT cap, 254-duration safety bound | Inferred constraints enforced by library |
| Flash persistence, RPC response/notification meanings, OTA/PIN | Unknown |

The sections below preserve the original technical research notes. Module numbers
identify private Android evidence; no module source is distributed. GATT write
success establishes delivery, not firmware semantic acceptance.

# PIX BLE protocol — Android 1.40

Источник: локальный `assets/index.android.bundle` из APK. Номера ниже — Metro module ID. iOS-декомпиляция уточнила частичные обновления и виджеты. Базовый показ, Live, текст и режимы проверены на устройстве: [отчёт](../hardware.md). Формат ответов RPC пока не декодирован.

## UUID

UUID строится как `0000XXXX` + один из суффиксов:

| Вариант | Суффикс |
|---|---|
| Original legacy | `-e984-11e7-b78e-ffd6fcc3450f` |
| Original V1 | `-e984-11e7-b78e-ffd6fcc34510` |
| Mini V1 | `-9552-4325-8021-a85f8136a5c4` |

| XXXX | Назначение |
|---|---|
| `0100` | основной сервис |
| `0101` | RPC request/response |
| `0103` | ширина, uint8 |
| `0104` | высота, uint8 |
| `0105` | максимум кадров, unsigned little-endian |
| `0106` | яркость, uint8, `floor(percent*255/100)` |
| `0107` | OTA enable |
| `0108` | OTA config |
| `0109` | OTA state |
| `0110` | demo state: 0=enabled, 1=disabled |

Device Information — стандартный `180A`, firmware `2A26`, hardware `2A27`. OTA здесь только документировано по именам в клиенте, команды обновления утилита не отправляет.

Модули 732 и 797 определяют UUID; 784 читает параметры. Для max frames helper 731 понимает значения длиной 1, 2 или 4 байта. Android распознаёт имя вида `Pix … aabbccddeeff`; CLI также учитывает advertised service UUID и позволяет явный выбор.

## RPC

Каждая запись в `0101` имеет вид:

```
call_id:u8 | opcode:u8 | payload...
```

Call ID начинается с 0 для нового объекта клиента, увеличивается по модулю 256. В показанной реализации нет добавочной длины, checksum, шифрования приложения или токена облачного сервера. Это не исключает системное BLE-сопряжение/шифрование соединения.

Android вызывает `BleManager.write(..., packet, packet.length)`, то есть write **with response**, передавая весь пакет; это видно в модулях 771 и 784. На Android запрашивается MTU 517 при разрешённой настройке и подходящей firmware. На Original V1 firmware 2.5.1 прошли полные записи кадра (326 байт) и части палитры из 160 цветов (483 байта).

| Opcode | Payload | Назначение |
|---|---|---|
| `00` | `frame_index:u16le, offset:u16le, indices[]` | кадр; исходный клиент всегда задаёт offset=0 |
| `01` | `start_color:u8, rgb_triplets[]` | общая палитра |
| `02` | `frame_count:u8` | число кадров |
| `03` | `offset:u16le, durations:u16le[]` | длительности в мс; клиент использует offset=0 |
| `04` | `direction:u8` | normal=0, alternate=1, normal-stop=2, alternate-stop=3 |
| `05` | пусто | сохранить анимацию в persistent memory |
| `06` | не исследован | print memory |
| `07` | `offset:u16le, config[]` | конфигурация режима |
| `08` | зависит от режима | input |
| `09` | пусто | restart |

В iOS `_setFramePartial` передаёт opcode 00 с ненулевым offset; `useLivePreview` использует offset=row*width, а FrameTestScreen — индекс одного пикселя. Offset кадра измеряется в пикселях; живое обновление подтверждено пользователем. Offset config измеряется в байтах. Ненулевой offset длительностей не исследован.

## Палитра и изображение

Модули 743 и 748: цвета извлекаются в порядке первого появления во всех кадрах; каждый цвет передаётся тремя байтами **R,G,B**. Номер цвета — его индекс, от 0 до 255. Палитра общая для всех кадров.

Модуль 747: `imageData[row][column]` превращается в линейный массив индексов строка за строкой. Кадр 16×20 содержит 320 байт. Полный RPC кадра — 326 байт с учётом call ID, opcode и двух uint16. Здесь нет упаковки нескольких пикселей в биты и нет RGB565.

Палитра разбивается исходным клиентом максимум на два пакета: первые 160 цветов (480 байт RGB) с start=0 и остаток с start=160. Размер максимального пакета первой части — 483 байта. Нельзя просто отправить эти 483 байта отдельными write по 20 байт: у каждой команды есть собственный заголовок.

Длительности — uint16 little-endian, миллисекунды. Например 300 мс → `2c 01`; 1000 мс → `e8 03`.

Go-библиотека дополнительно ограничивает каждое значение GATT 512 байтами: до 506 пикселей в одной записи и до 254 длительностей в одном пакете. Действует и меньший предел кадров, прочитанный с устройства. Частичное обновление пикселей реализовано в SetFrame и Live; разделение массива длительностей на пакеты пока не реализовано.

## Загрузка

Модуль 739 выполняет:

1. `07`, payload `00 00 00`: config offset=0, mode NONE=0.
2. `02 00`: сбросить число кадров.
3. `01`: записать палитру.
4. `00`: записать каждый кадр с его индексом.
5. `03`: записать длительности всех кадров.
6. `04`: задать направление.
7. `02 N`: установить число кадров.
8. `05`: сохранить, если выбран play-on-boot.

Отдельного включения MODE_ANIMATION после установки числа кадров в этой функции нет. Go повторяет этот порядок. Показ без сохранения подтверждён на firmware 2.5.1.

Константы режимов из 732: NONE=0, ANIMATION=1, SCROLLING=2, OTA_PROGRESS=3, CLOCK=6, BIKE=7, COUNTDOWN=8, STOPWATCH=9, PIX_BLOCKS=16, PIXEL_BREAKER=17, CRAWLER=18. Payload виджетов и игр включён в библиотеку; см. таблицу ниже и [API](../api.md).

## Config виджетов

Перед config добавляется `call_id, 07, offset:u16le=0`.

| Режим | Config |
|---|---|
| Bike | `07, stage:u8, 00`, init=0/idle=1/stop=2/left=3/right=4 |
| Countdown | `08, 00, seconds:u16le, 05`, 0..5999 секунд |
| Stopwatch | `09, 00, 01, 00, 00, ff ff ff ff ff ff ff ff` |
| Crawler | `12, 00` |
| Pix Blocks | `10, step_ms:u16le, 00` |
| Pixel Breaker | `11, 00` |
| Scrolling text | `02, direction=0, speed_ms:u16le, char_width=8, char_height=20, length:u16le, sprite_indices[]` |

Текстовые спрайты — обычные кадры с glyph 8×20 в левой части экрана, остальное чёрное. Палитра: 0 — чёрный, 1 — foreground. После спрайтов и config задаётся число спрайтов opcode 02.

Clock config: `06, local_epoch:u32le, alarm_epoch:u32le, alarm_duration:u8, stage:u8, watch_face:u8, format:u8, position:u16le, seven_colour_indices[], timezone_cstr, 00`. iOS вычисляет local_epoch как Unix epoch + локальный UTC offset. При отключённом будильнике alarm_epoch=local_epoch-60, duration=0. API повторяет iOS с IANA timezone string; Europe/Moscow подтверждён пользователем.

Stopwatch отличается от старого Android: в iOS восемь `ff`, в Android четыре элемента 65535 после flatten превращаются в четыре байта. Go использует iOS-формат, принятый устройством.

## Игровой ввод

`call_id, 08, input_id:u8, button:u8`. Input ID начинается с 1, идёт до 255 и возвращается к 1; он независим от call ID. Button up=1/down=2/left=3/right=4/action=5. В Stopwatch start=1/stop=2/reset=3/lap=4. Логика встроенных игр и отрисовка живут в firmware; хост включает режим и отправляет события.

## Границы достоверности

Структура команд подтверждена Android и iOS; живой кадр, базовый показ, текст и несколько встроенных режимов — аппаратно и визуально. Устройство: firmware 2.5.1 / hardware 3.2 / 16×20. MaxFrameCount вернул 2048, но count payload в обоих приложениях остаётся uint8. Ответы RPC, персистентность после перезапуска и другие firmware не проверены: [подробный отчёт](../hardware.md).


## Packet offsets and synthetic example

| Offset | Length | Meaning |
|---|---|---|
| 0 | 1 | call ID, wraps modulo 256 |
| 1 | 1 | opcode |
| 2 | variable | opcode-specific payload (tables above) |

Synthetic full frame command `03 00 00 00 00 00 00 01 02 00` means call 3,
opcode 0, frame index 0 (u16le), pixel offset 0 (u16le), four palette indices
0/1/2/0. Durations 300/1000 encode as `2c 01 e8 03`. Tests in
`pix/protocol_test.go` assemble a complete synthetic 2×2 upload.

## Connection and access

Scan by PIX name or a known advertised service, connect, discover service 0100,
resolve RPC 0101, read dimensions 0103/0104, capacity 0105 and brightness 0106.
Read Device Information strings if available. No additional application-level
initialization handshake is established. The uploader explicitly sets mode NONE
and resets frame count before writing palette/frames. Writes use response;
0101 notifications are optional and their payload remains unknown. 0106 and 0110
can be written directly. OTA characteristic access flags are not established.

## Interchange is separate

`.pixpack` is JSON (optionally gzip), not a packet sent directly to the device.
The Go reader converts colors, durations, direction and custom glyphs into RPC.
No third party collection, font or specification text is bundled.
