package media

import (
	"context"
	"github.com/badoux/goscraper"
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

// GoscraperFetchPreview fetches a preview of a URL using goscraper.
// https://github.com/badoux/goscraper
func GoscraperFetchPreview(ctx context.Context, url string) (*LinkPreview, error) {
	s, err := goscraper.Scrape(url, 5)
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
