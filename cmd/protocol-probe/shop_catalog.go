package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var defaultShopINIPath = filepath.Join(defaultModernShareRoot, "trade", "shop.ini")

// shopCatalogItem is one retail listing from the modern share/trade/shop.ini
// section. The file intentionally repeats the key "0", so a conventional INI
// reader would lose all but the final item; scan the selected section line by
// line instead.
type shopCatalogItem struct {
	configID    string
	amount      int32
	capitalType int32
	price       int32
	page        int32
	position    int32
}

func loadShopCatalogSection(path, shopID string) ([]shopCatalogItem, int32, int32, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("open shop catalog %s: %w", path, err)
	}
	defer file.Close()

	wantedHeader := "[" + shopID + "]"
	inSection := false
	shopType := int32(0)
	maxPage := int32(1)
	items := make([]shopCatalogItem, 0, 10)
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			if inSection {
				break
			}
			inSection = line == wantedHeader
			continue
		}
		if !inSection {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "Type" {
			parsed, parseErr := strconv.ParseInt(value, 10, 32)
			if parseErr != nil {
				return nil, 0, 0, fmt.Errorf("shop %s line %d: invalid Type %q: %w", shopID, lineNumber, value, parseErr)
			}
			shopType = int32(parsed)
			continue
		}
		// Item rows use an intentionally repeated numeric key (normally 0).
		if _, parseErr := strconv.ParseInt(key, 10, 32); parseErr != nil {
			continue
		}
		parts := strings.Split(value, ",")
		if len(parts) < 7 {
			return nil, 0, 0, fmt.Errorf("shop %s line %d: expected at least 7 item columns, got %d", shopID, lineNumber, len(parts))
		}
		parseColumn := func(column int, name string) (int32, error) {
			parsed, parseErr := strconv.ParseInt(strings.TrimSpace(parts[column]), 10, 32)
			if parseErr != nil {
				return 0, fmt.Errorf("shop %s line %d: invalid %s %q: %w", shopID, lineNumber, name, parts[column], parseErr)
			}
			return int32(parsed), nil
		}
		amount, parseErr := parseColumn(1, "amount")
		if parseErr != nil {
			return nil, 0, 0, parseErr
		}
		capitalType, parseErr := parseColumn(2, "capital type")
		if parseErr != nil {
			return nil, 0, 0, parseErr
		}
		price, parseErr := parseColumn(3, "price")
		if parseErr != nil {
			return nil, 0, 0, parseErr
		}
		page, parseErr := parseColumn(5, "page")
		if parseErr != nil {
			return nil, 0, 0, parseErr
		}
		position, parseErr := parseColumn(6, "position")
		if parseErr != nil {
			return nil, 0, 0, parseErr
		}
		if page < 1 || position < 0 {
			return nil, 0, 0, fmt.Errorf("shop %s line %d: invalid page=%d position=%d", shopID, lineNumber, page, position)
		}
		items = append(items, shopCatalogItem{
			configID: strings.TrimSpace(parts[0]), amount: amount, capitalType: capitalType,
			price: price, page: page, position: position,
		})
		if page > maxPage {
			maxPage = page
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, 0, 0, fmt.Errorf("read shop catalog %s: %w", path, err)
	}
	if !inSection {
		return nil, 0, 0, fmt.Errorf("shop catalog %s has no section %q", path, shopID)
	}
	if len(items) == 0 {
		return nil, 0, 0, fmt.Errorf("shop catalog %s section %q has no items", path, shopID)
	}
	return items, shopType, maxPage, nil
}
