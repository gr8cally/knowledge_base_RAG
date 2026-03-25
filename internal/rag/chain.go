package rag

import (
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/schema"
)

func NewConversationChain(
	llm llms.Model,
	retriever schema.Retriever,
	history schema.ChatMessageHistory,
	maxTurns int,
) chains.ConversationalRetrievalQA {
	mem := memory.NewConversationWindowBuffer(
		maxTurns,
		memory.WithChatHistory(history),
		memory.WithInputKey("question"),
		memory.WithOutputKey("text"),
	)

	chain := chains.NewConversationalRetrievalQAFromLLM(llm, retriever, mem)
	chain.ReturnSourceDocuments = true
	return chain
}
