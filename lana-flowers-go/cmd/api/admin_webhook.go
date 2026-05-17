package main

import (
	"fmt"
	"os"

	"lana-flowers-go/internal/adminbot"
)

// runAdminWebhookCmd — одноразовая настройка webhook'а для админ-бота
// в Telegram. URL и secret берутся из env (ADMIN_WEBHOOK_URL / ADMIN_WEBHOOK_SECRET).
//
// Usage: ./app admin-webhook
//
// После успеха Telegram начнёт слать POST'ы на URL с заголовком
// X-Telegram-Bot-Api-Secret-Token = ADMIN_WEBHOOK_SECRET.
func runAdminWebhookCmd(_ []string) {
	url := os.Getenv("ADMIN_WEBHOOK_URL")
	if url == "" {
		fmt.Fprintln(os.Stderr, "ADMIN_WEBHOOK_URL is required in env (e.g. https://your.host/tg/admin-webhook)")
		os.Exit(2)
	}
	secret := os.Getenv("ADMIN_WEBHOOK_SECRET")
	if secret == "" {
		fmt.Fprintln(os.Stderr, "ADMIN_WEBHOOK_SECRET is required in env")
		os.Exit(2)
	}
	if os.Getenv("ADMIN_BOT_TOKEN") == "" {
		fmt.Fprintln(os.Stderr, "ADMIN_BOT_TOKEN is required in env")
		os.Exit(2)
	}

	if err := adminbot.SetWebhook(url, secret); err != nil {
		fmt.Fprintf(os.Stderr, "setWebhook failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ admin webhook set to %s\n", url)
}
