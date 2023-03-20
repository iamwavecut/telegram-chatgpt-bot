# Simple Telegram bot integration to OpenAI ChatGPT API

---
## Disclaimer 
> ⚠️ This code represents a working instance of the bot named [@net_nebot on Telegram](https://t.me/net_nebot). Other bots may be running the same code, and may be used for malicious purposes. Use at your own risk!

> This is not an official OpenAI nor Telegram product. This is a community project.

> The code is provided as is, and is not guaranteed to work.
---
## Self-hosting
The easiest way to self-host the bot is to build a Docker image and run it on a server of your choice. The Dockerfile is provided in the repository.

You will need to provide the following arguments to the build command:
```shell
docker build -t telegram-chatgpt-bot . --build-arg OPENAI_TOKEN=<required, your_api_token> --build-arg BOT_TOKEN=<required, your_bot_token> --build-arg CHATGPT_VERSION=<optional, 3.5 | 4> 
```

You can also run the bot locally
```shell
docker run -d --restart always --name  telegram-chatgpt-bot telegram-chatgpt-bot
```
---
## Translations
The bot is currently available in the following languages: English, Russian, Belarusian, Ukrainian. Feel free to contribute translations for other languages! The ChatGPT API itself understands a lot more languages, so go give it a try!
