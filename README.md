# Simple Telegram bot integration to OpenAI ChatGPT API

---
### Disclaimer 
> ⚠️ This code represents a working instance of the bot named [@net_nebot on Telegram](https://t.me/net_nebot). Other bots may be running the same code, and may be used for malicious purposes. Use at your own risk!

> This is not an official OpenAI nor Telegram product. This is a community project.

> The code is provided as is, and is not guaranteed to work.
---
### Self-hosting
The easiest way to self-host the bot is to build a Docker image and run it on a server of your choice. The Dockerfile is provided in the repository.

You will need to provide the following arguments to the build command:
```shell
docker build -t net-nebot . --build-arg OPENAI_API_KEY=<your_api_key> --build-arg TELEGRAM_BOT_TOKEN=<your_bot_token>
```

You can also run the bot locally
```shell
docker run -d --name net-nebot net-nebot
```
---
# Translations
The bot is currently available in the following languages: English, Russian, Belarusian, Ukrainian. Feel free to contribute translations for other languages!