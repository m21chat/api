package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/cors"
)

// Interfaces to match the TypeScript structure
type OllamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OllamaChatPrompt struct {
	Model    string          `json:"model"`
	Messages []OllamaMessage `json:"messages"`
}

// Replicate the generateChat function and associated types
type ChatResponse struct {
	Response *http.Response `json:"response"`
}

func generateChat(prompt OllamaChatPrompt) (*ChatResponse, error) {
	// Get the OLLAMA_BASE_URL from environment variables
	baseURL := os.Getenv("OLLAMA_BASE_URL")
	if baseURL == "" {
		baseURL = "http://192.168.0.37:11434/api" // Default if not specified
	}

	fmt.Println(prompt)

	prompt.Model = "solar"

	fmt.Println(prompt)

	// Marshal the prompt into JSON
	jsonData, err := json.Marshal(prompt)
	if err != nil {
		return nil, err
	}

	// Create a new HTTP request
	req, err := http.NewRequest("POST", baseURL+"/chat", io.NopCloser(strings.NewReader(string(jsonData))))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/x-ndjson")
	req.Header.Set("User-Agent", "M21")

	client := &http.Client{Timeout: time.Second * 10} // Example timeout
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return &ChatResponse{Response: resp}, nil
}

func main() {
	router := gin.Default() // Create a Gin router

	// Configure CORS middleware
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://web01.muthur.net", "http://localhost:3000"}, // Replace with your frontend origins
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
		AllowCredentials: true, // Optional, to send cookies for authentication
	})

	router.Use(func(ctx *gin.Context) {
		c.HandlerFunc(ctx.Writer, ctx.Request)
		ctx.Next()
	})

	// API Endpoint using your generateChat function
	router.POST("/chat", func(c *gin.Context) {
		var prompt OllamaChatPrompt
		if err := c.BindJSON(&prompt); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Error parsing JSON"})
			return
		}
		fmt.Println("Prompt:", prompt)

		resp, err := generateChat(prompt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating response"})
			return
		}
		defer resp.Response.Body.Close()
		// Directly forward headers
		for header, values := range resp.Response.Header {
			for _, value := range values {
				c.Header(header, value)
			}
		}
		// Set up for streaming
		c.Status(resp.Response.StatusCode)
		c.Writer.Header().Set("Content-Type", "text/event-stream") // Or appropriate streaming content type
		c.Writer.Flush()

		// Stream the data
		reader := bufio.NewReader(resp.Response.Body)
		for {
			chunk, err := reader.ReadBytes('\n') // Adjust delimiter if necessary
			if err != nil {
				if err != io.EOF {
					// Handle errors other than end of stream
				}
				break
			}
			c.Writer.Write(chunk)
			c.Writer.Flush()
		}
	})

	router.Run(":8080") // Run the server (adjust port if needed)
}
