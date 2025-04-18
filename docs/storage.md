# Устройство Storage

## API-интерфейс

Storage предоставляет три основных метода API:
- `POST /storage/upload` — загрузка файла
- `GET /storage/get` — получение файла
- `DELETE /storage/remove` — удаление файла

Все API-методы принимают следующие параметры:
- `id` — уникальный идентификатор объекта (например, problem ID или submission ID)
- `dataType` — тип данных (например, `problem` или `submission`)
- `filepath` — путь к файлу или имя файла

## Filesystem

Filesystem представляет собой интерфейс для работы с файловой системой и имеет следующие методы:
- `SaveFile` — сохранение загруженного файла
- `RemoveFile` — удаление файла
- `GetFilePath` — получение пути к файлу

## StorageConnector

Для взаимодействия с Storage из других сервисов используется `StorageConn`, который предоставляет следующие методы:
- `Download` — загрузка файла из Storage
- `Upload` — отправка файла в Storage
- `Delete` — удаление файла из Storage
