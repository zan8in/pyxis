package favicon

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/zan8in/pyxis/pkg/http/retryhttpclient"
	"github.com/zan8in/pyxis/pkg/util/faviconhashutil"
	"github.com/zan8in/pyxis/pkg/util/stringutil"
)

func HandleFaviconHash(target, body string) (string, error) {
	if strings.HasSuffix(target, ".ico") {
		return doFaviconHash(target), nil
	}

	potentialURLs, err := extractPotentialFavIconsURLs(body)
	if err != nil {
		return "", err
	}

	u, err := url.Parse(target)
	if err != nil {
		return "", err
	}
	target = u.Scheme + "://" + u.Host

	faviconPath := "/favicon.ico"

	if len(potentialURLs) > 0 {
		for _, potentialURL := range potentialURLs {
			if len(potentialURL) > 0 {
				if strings.HasPrefix(potentialURL, "//") {
					faviconPath = "http:" + potentialURL
					return faviconPath, nil
				} else if strings.HasPrefix(potentialURL, "http") {
					faviconPath = potentialURL
					return faviconPath, nil
				} else if strings.HasSuffix(potentialURL, ".ico") ||
					strings.HasSuffix(potentialURL, ".png") ||
					strings.HasSuffix(potentialURL, ".jpg") {
					faviconPath = target + "/" + strings.Trim(potentialURL, "/")
					return faviconPath, nil
				}
			}
		}
	}

	return target + faviconPath, nil
}

func extractPotentialFavIconsURLs(body string) ([]string, error) {
	var potentialURLs []string
	document, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	document.Find("link").Each(func(i int, item *goquery.Selection) {
		href, okHref := item.Attr("href")
		rel, okRel := item.Attr("rel")
		isValidRel := okRel && stringutil.EqualFoldAny(rel, "icon", "shortcut icon", "mask-icon", "apple-touch-icon")
		if okHref && isValidRel {
			potentialURLs = append(potentialURLs, href)
		}
	})
	return potentialURLs, nil
}

func FaviconHash(target, body string) string {
	// HandleFaviconHash(target, body)
	if target == "" {
		return ""
	}
	if body == "" {
		return ""
	}
	if url, err := HandleFaviconHash(target, body); err == nil && len(url) > 0 {
		return doFaviconHash(url)
	}
	return ""
}

func doFaviconHash(url string) string {
	result, err := retryhttpclient.GetHttpRequest(url)
	if err != nil {
		return ""
	}
	if result.StatusCode == 200 && len(result.Body) > 0 {
		hashNum, err := faviconhashutil.FaviconHash([]byte(result.Body))
		if err == nil {
			return fmt.Sprintf("%d", hashNum)
		}
	}
	return ""
}
