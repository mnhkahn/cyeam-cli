package news

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

func ExtractImage(pageURL string) string {
	if strings.Contains(pageURL, "mp.weixin.qq.com") {
		return ""
	}

	resp, err := httpClient.Get(pageURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return ""
	}

	// 1. og:image
	if ogImage, exists := doc.Find(`meta[property="og:image"]`).Attr("content"); exists {
		return toAbsoluteURL(pageURL, ogImage)
	}

	// 2. twitter:image
	if twitterImage, exists := doc.Find(`meta[name="twitter:image"]`).Attr("content"); exists {
		return toAbsoluteURL(pageURL, twitterImage)
	}

	// 3. 正文第一张图
	var firstImage string
	doc.Find("article img, main img, .content img, .post img").EachWithBreak(func(i int, s *goquery.Selection) bool {
		if src, exists := s.Attr("src"); exists && isValidImage(src) {
			firstImage = toAbsoluteURL(pageURL, src)
			return false
		}
		return true
	})
	if firstImage != "" {
		return firstImage
	}

	// 4. 页面第一张图兜底
	doc.Find("img").EachWithBreak(func(i int, s *goquery.Selection) bool {
		if src, exists := s.Attr("src"); exists && isValidImage(src) {
			firstImage = toAbsoluteURL(pageURL, src)
			return false
		}
		return true
	})

	return firstImage
}

func toAbsoluteURL(base, src string) string {
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		return src
	}
	parsedBase, err := url.Parse(base)
	if err != nil {
		return src
	}
	parsedSrc, err := parsedBase.Parse(src)
	if err != nil {
		return src
	}
	return parsedSrc.String()
}

func isValidImage(src string) bool {
	srcLower := strings.ToLower(src)

	// 排除 SVG
	if strings.HasSuffix(srcLower, ".svg") {
		return false
	}

	// 排除已知小图片 URL
	keywords := []string{
		"avatar", "profile", "gravatar", "userpic", "headshot",
		"badge", "shield", "icon", "logo",
		"ci/", "build", "status", "workflow",
	}
	for _, k := range keywords {
		if strings.Contains(srcLower, k) {
			return false
		}
	}

	return true
}
