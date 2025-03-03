package media

import (
	"context"
	"github.com/devlikeapro/goscraper"
	"net/url"
	"regexp"
	"strings"
)

var UrlRegex = `(http(s)?:\/\/.)(www\.)?[-a-zA-Z0-9@:%._\+~#=]{2,256}\.[a-z]{2,6}\b([-a-zA-Z0-9@:%_\+.~#?&//=]*)`
var UrlRe = regexp.MustCompile(UrlRegex)

func ExtractUrlFromText(text string) string {
	match := UrlRe.FindString(text)
	return match
}

func MakeSureURL(text string) string {
	var url string
	if !strings.HasPrefix(text, "http") || !strings.HasPrefix(text, "https") {
		url = "https://" + text
	} else {
		url = text
	}
	return url
}

type LinkPreview struct {
	Url         string
	Title       string
	Description string
	ImageUrl    string
	IconUrl     string
}

var ScrapeHeaders = map[string]string{
	"User-Agent": "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
	"Accept":     "Mozilla/5.0 (Windows; Windows NT 6.3; Win64; x64) Gecko/20100101 Firefox/67.7",
}

// GoscraperFetchPreview fetches a preview of a URL using goscraper.
// https://github.com/devlikeapro/goscraper
func GoscraperFetchPreview(ctx context.Context, uri string) (*LinkPreview, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	scraper := goscraper.Scraper{
		Url:         u,
		MaxRedirect: 5,
		Headers:     ScrapeHeaders,
	}
	s, err := scraper.Scrape(ctx)
	if err != nil {
		return nil, err
	}

	var image string
	if len(s.Preview.Images) > 0 {
		image = s.Preview.Images[0]
	}
	preview := &LinkPreview{
		Url:         s.Preview.Link,
		Title:       s.Preview.Title,
		Description: s.Preview.Description,
		ImageUrl:    image,
		IconUrl:     s.Preview.Icon,
	}
	return preview, nil
}
