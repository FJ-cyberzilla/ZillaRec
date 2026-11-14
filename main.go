package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strings"
	"yt-termux/config"
	"yt-termux/downloader"
	"yt-termux/network"
)

func main() {
	// 1. Load application configuration & print active settings
	cfg := config.LoadDefaultConfig()
	cfg.PrintConfig()

	// 2. Initialize custom Quad9 transport check
	transport := network.NewQuad9Transport()
	client := &http.Client{Transport: transport}

	resp, err := client.Get("https://www.youtube.com")
	if err != nil {
		fmt.Printf("[Warning] Network check notice: %v\n", err)
	} else {
		resp.Body.Close()
		fmt.Println("[Status] Secure connection established via Quad9 DNS.")
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("========================================")
	fmt.Println("    TERMUX GOLANG HIGH-SPEED YOUTUBE    ")
	fmt.Println("========================================")
	
	fmt.Print("Enter YouTube Video or Short URL: ")
	urlInput, _ := reader.ReadString('\n')
	urlInput = strings.TrimSpace(urlInput)

	if urlInput == "" {
		fmt.Println("[Error] URL cannot be empty.")
		return
	}

	// 3. Present quality selection menu using profiles from config module
	profiles := config.GetQualityProfiles()
	formatSpec := downloader.PromptQualityWithConfig(profiles)

	// 4. Resolve cookie path securely from config settings
	cookieFile := cfg.CookieFile
	if _, err := os.Stat(cookieFile); os.IsNotExist(err) {
		cookieFile = "" // Optional fallback if cookies.txt is missing
	}

	// 5. Execute download process
	err = downloader.ExecuteDownload(urlInput, formatSpec, cookieFile)
	if err != nil {
		fmt.Printf("[Error] Download failed: %v\nVerify yt-dlp and ffmpeg are installed in Termux.\n", err)
	} else {
		fmt.Println("\n[Success] Video downloaded successfully!")
	}
}
