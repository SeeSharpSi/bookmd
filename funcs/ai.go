package funcs

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"

	"github.com/sashabaranov/go-openai"
)

// ConvertImageToMarkdown takes a file path,
// sends the image to the AI, and returns the markdown transcription.
func ConvertImageToMarkdown(ctx context.Context, client *openai.Client, imagePath string) (string, error) {
	// Initialize OpenAI client if not provided
	if client == nil {
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return "", fmt.Errorf("OPENAI_API_KEY environment variable not set")
		}
		client = openai.NewClient(apiKey)
	}
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read image file: %w", err)
	}

	mimeType := http.DetectContentType(imageData)

	base64Image := base64.StdEncoding.EncodeToString(imageData)

	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Image)

	req := openai.ChatCompletionRequest{
		Model: "gemini-3-flash-preview",
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleUser,
				MultiContent: []openai.ChatMessagePart{
					{
						Type: openai.ChatMessagePartTypeText,
						Text: "Transcribe this image of notes into clean Markdown. Use headers, bullet points, and code blocks to match the visual structure.",
					},
					{
						Type: openai.ChatMessagePartTypeImageURL,
						ImageURL: &openai.ChatMessageImageURL{
							URL: dataURL,
						},
					},
				},
			},
		},
	}

	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("ai request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	return resp.Choices[0].Message.Content, nil
}
