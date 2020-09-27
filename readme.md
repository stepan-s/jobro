# Менеждер периодических и постоянных процессов

Типовое применение - запуск крон заданий и обработчиков очередей в докер-контейнере.

Постановка задачи:
* Запускать периодические задания
* Поддерживать постоянных обработчиков в требуемом количестве
* Предоставлять статистику для prometheus
* Предоставлять HTTP API для получения информации и ручного запуска задач
* Перезагружать конфигурацию по сигналу (`SIGUSR1`)
* Уведомлять обработчики о необходимости остановки (`SIGINT`)

### Пример запуска

```bash
jobro \
  --addr=localhost:8080 \
  --config-command="cat a_config.json" \
  --shutdown-timeout=300 \
  --log-level=8
```

Конфиг должна возвращать какая-то команда, в примере выше это `cat` возвращающий содержимое файла, можно использовать `curl` или скрипт динамически формирующий или конвертирующий конфигурацию.

### Пример конфига:

```json
{
  "schedule": [
    {"cron":  "0 * * * * *", "cmd":  "echo 'every minute'", "group":  "anything"},
    {"cron":  "0 0 * * * *", "cmd":  "echo 'every hour'", "group":  "anything"},
    {"cron":  "manual", "cmd":  "echo 'can start by api call'", "group":  "anything"}
  ],
  "instant": [
    {"cmd":  "/opt/a_worker", "count":  5, "group":  "anything"}
  ]
}
```

Перезагрузка конфигурации:

```bash
kill -s USR1 $jobro_pid 
```

### Запросы к API

`http://localhost:8080/metrics` - prometheus

`http://localhost:8080/api/info` - информация о задачах

`http://localhost:8080/api/schedule/run?id=task_uuid` - внеочередной запуск периодического, либо `manual` задания
