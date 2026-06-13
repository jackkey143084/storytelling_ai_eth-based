README.md (ключевые шаги)
Подготовка:
Скопировать проект в каталог.
Убедиться, что Docker и Docker Compose установлены.
В go-service/keys/dev_account.key положить тестовый приватный ключ в hex (без 0x).
(Опционально) в infra/hardhat заменить deploy скрипт и скомпилировать контракт для получения ABI и адреса.
Запуск:
docker-compose up --build
Когда geth поднят (RPC на localhost:8545), скомпилируйте и деплойте контракт:
Откройте контейнер hardhat (или run npx hardhat in infra/hardhat) и выполните npx hardhat run --network localgeth scripts/deploy.js
Скопируйте адрес контракта и установите переменную CONTRACT_ADDRESS в go-service (можно через docker-compose override env или REST call to /set_contract — в этом шаблоне вы можете просто прописать CONTRACT_ADDRESS в docker-compose env).
Сгенерируйте рассказ Rust‑сервисом, загрузите текст в IPFS (HTTP API http://localhost:5001), получите CID.
Вызовите Go‑сервис: POST http://localhost:8081/mint с JSON: { "to_address":"0x...", "ipfs_cid":"Qm...", "checkpoint":"<64 hex chars>" }
Go‑сервис подпишет и отправит транзакцию в локальный geth; после майнинга (dev mode майнит автоматически) транзакция будет включена.
Примечания безопасности:
Приватный ключ хранится в контейнере в открытом виде для удобства тестирования только в локальной среде.
Для продакшена используйте Hardware Wallets, KMS, безопасное хранение ключей и проверяйте все входные данные.

Дополнительные  доработки (naprimer):


- Полный, компилируемый Go‑код с корректными импортами, генерацией и использованием
- сгенерированных Go‑bindings (abigen) для StoryNFT (чтобы не делать low-level ABI packing).
- 
- Скрипты Hardhat для автоматического деплоя и сохранения адреса контракта в go-service/.env.
- 
 * Полную интеграцию Rust клиента: после генерации текста автоматически загрузка в IPFS и вызов mint.
 * 
Пример end‑to‑end: generate -> ipfs upload -> mint -> получить tx hash -> получить tokenURI.
