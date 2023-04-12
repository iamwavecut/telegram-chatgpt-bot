package consts

const (
	StrHello = "Hi, my name is %s!"
	StrIntro = "I'm a bot, based on ChatGPT API. My source code " +
		"is available %s, for you to audit it. I do not " +
		"log nor store your message, but keep in mind, that " +
		"I should store some history at runtime, to keep context, " +
		"and I send it to OpenAI API. If it's concerning " +
		"you – please stop and delete me."
	StrOutro = "If you want to restart the conversation from " +
		"scratch, just type /start and the bot's " +
		"recent memories will fade away."
	StrTimeout = "I'm sorry, but this takes an unacceptable " +
		"duration of time to answer. Request aborted."
	StrNoPublic = "Unfortunately, I work terrible in groups, " +
		"as ChatGPT was designed to be used in dialogues. " +
		"Please message me in private."
	StrRequestError = "Unfortunately, there was an error during the request. " +
		"Please try again later. If it doesn't help, " +
		"please contact the maintainer."
)
