package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
	"github.com/gorilla/websocket"
)

func (s *Scraper) scrollToTop(ctx context.Context, retries int) error {
	var currentScrollY float64

	for i := 0; i <= retries; i++ {
		// Use chromedp to scroll to the top of the page
		err := chromedp.Run(ctx,
			chromedp.KeyEvent(kb.Home),
			chromedp.Sleep(900*time.Millisecond),
			chromedp.KeyEvent(kb.PageUp),
			chromedp.Evaluate(`Math.round(window.scrollY)`, &currentScrollY),
		)
		if err != nil {
			if i == retries {
				fmt.Println("Failed to scroll back to the top:", err)
				return err
			}
			continue
		}

		if int(currentScrollY) == 0 {
			return nil // Successfully scrolled to the top
		}

		fmt.Printf("Retry %d: Scroll position is not exactly at the top, current Y position: %d\n", i+1, int(currentScrollY))
	}

	return fmt.Errorf("unable to scroll to the top after %d retries", retries)
}

func (s *Scraper) determineHeight(ctx context.Context) (int, error) {
	var lastScrollY float64

	// Scroll to the bottom of the page to capture the scroll height
	err := chromedp.Run(ctx,
		chromedp.KeyEvent(kb.End),
		chromedp.Sleep(900*time.Millisecond),
		chromedp.KeyEvent(kb.PageDown),
		chromedp.Evaluate(`Math.round(window.scrollY)`, &lastScrollY),
	)
	if err != nil {
		fmt.Println("Failed to scroll to the bottom:", err)
		return 0, err
	}

	scrollYInt := int(lastScrollY)
	fmt.Println("last scroll: ", scrollYInt)

	retries := 2
	err = s.scrollToTop(ctx, retries)
	if err != nil {
		return 0, err
	}

	var currentScrollY float64
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`Math.round(window.scrollY)`, &currentScrollY),
	)
	if err != nil {
		fmt.Println("Failed to retrieve current scroll position after scrolling to the top:", err)
		return 0, err
	}
	currentScrollYInt := int(currentScrollY)
	fmt.Println("current scroll y: ", currentScrollYInt)

	if currentScrollYInt > 0 {
		fmt.Println("Note: Scroll position is not exactly at the top, current Y position:", currentScrollYInt)
	}

	return scrollYInt, nil
}

func (s *Scraper) cacheDataInRedis(userId string, url string, screenshots []string, market string, audience string, insights string) {
	cachedData := CachedData{
		Screenshots: screenshots,
		Market:      market,
		Audience:    audience,
		Insights:    insights,
	}
	jsonData, err := json.Marshal(cachedData)
	if err != nil {
		log.Printf("Error marshaling cached data: %v", err)
		return
	}

	key := userId + ":" + url
	_, err = s.RedisClient.Set(context.Background(), key, jsonData, 24*time.Hour).Result()
	if err != nil {
		log.Printf("Error saving cached data to Redis: %v", err)
	}
}

func (s *Scraper) sendWebSocketMessage(conn *websocket.Conn, msg WebSocketMessage) {
	message, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal WebSocket message: %v", err)
		return
	}
	if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
		log.Printf("Failed to send WebSocket message: %v", err)
	}
}
