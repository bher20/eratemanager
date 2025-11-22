package rates

import (
    "errors"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "regexp"
    "sort"
    "strings"
    "time"
)

// PDFDiscoveryTimeout controls how long we wait for the landing page.
var PDFDiscoveryTimeout = 10 * time.Second

// DiscoverPDFURL fetches the provider's landing page and discovers the best PDF URL.
func DiscoverPDFURL(p ProviderDescriptor) (string, error) {
    if p.LandingURL == "" {
        return "", fmt.Errorf("provider %q has no LandingURL", p.Key)
    }

    client := &http.Client{Timeout: PDFDiscoveryTimeout}
    resp, err := client.Get(p.LandingURL)
    if err != nil {
        return "", fmt.Errorf("fetch landing url: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return "", fmt.Errorf("landing url returned status %d", resp.StatusCode)
    }

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", fmt.Errorf("read landing body: %w", err)
    }

    return discoverPDFURLFromHTML(p.LandingURL, string(body))
}

func discoverPDFURLFromHTML(baseURL, html string) (string, error) {
    base, err := url.Parse(baseURL)
    if err != nil {
        return "", fmt.Errorf("parse base url: %w", err)
    }

    type candidate struct {
        rawHref string
        text    string
        score   int
    }

    var candidates []candidate

    // Anchor tags with link text
    anchorRe := regexp.MustCompile(`(?is)<a[^>]+href="([^"]+\.pdf)"[^>]*>([^<]*)</a>`)
    for _, m := range anchorRe.FindAllStringSubmatch(html, -1) {
        href := strings.TrimSpace(m[1])
        text := strings.TrimSpace(htmlUnescape(m[2]))
        score := scorePDFCandidate(href, text)
        candidates = append(candidates, candidate{rawHref: href, text: text, score: score})
    }

    // Fallback: any href="...pdf"
    if len(candidates) == 0 {
        hrefRe := regexp.MustCompile(`(?i)href="([^"]+\.pdf)"`)
        for _, m := range hrefRe.FindAllStringSubmatch(html, -1) {
            href := strings.TrimSpace(m[1])
            score := scorePDFCandidate(href, "")
            candidates = append(candidates, candidate{rawHref: href, text: "", score: score})
        }
    }

    if len(candidates) == 0 {
        return "", errors.New("no PDF links found on page")
    }

    sort.SliceStable(candidates, func(i, j int) bool {
        if candidates[i].score != candidates[j].score {
            return candidates[i].score > candidates[j].score
        }
        iHTTPS := strings.HasPrefix(strings.ToLower(candidates[i].rawHref), "https://")
        jHTTPS := strings.HasPrefix(strings.ToLower(candidates[j].rawHref), "https://")
        if iHTTPS != jHTTPS {
            return iHTTPS
        }
        return candidates[i].rawHref < candidates[j].rawHref
    })

    best := candidates[0].rawHref
    bestURL, err := base.Parse(best)
    if err != nil {
        return "", fmt.Errorf("resolve href %q: %w", best, err)
    }

    return bestURL.String(), nil
}

func scorePDFCandidate(href, text string) int {
    hrefLower := strings.ToLower(href)
    textLower := strings.ToLower(text)

    score := 0

    if strings.Contains(textLower, "residential") {
        score += 5
    }
    if strings.Contains(textLower, "rate") || strings.Contains(textLower, "schedule") {
        score += 3
    }
    if strings.Contains(hrefLower, "residential") {
        score += 3
    }
    if strings.Contains(hrefLower, "rates") || strings.Contains(hrefLower, "rs") {
        score += 2
    }
    if strings.Contains(textLower, "current") || strings.Contains(hrefLower, "2025") {
        score += 1
    }

    return score
}

func htmlUnescape(s string) string {
    replacer := strings.NewReplacer(
        "&amp;", "&",
        "&lt;", "<",
        "&gt;", ">",
        "&quot;", `"`,
        "&#39;", "'",
    )
    return replacer.Replace(s)
}

// RefreshProviderPDF discovers and downloads the provider PDF into DefaultPDFPath.
func RefreshProviderPDF(p ProviderDescriptor) (string, error) {
    pdfURL, err := DiscoverPDFURL(p)
    if err != nil {
        return "", err
    }

    client := &http.Client{Timeout: 30 * time.Second}
    resp, err := client.Get(pdfURL)
    if err != nil {
        return "", fmt.Errorf("download pdf: %w", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return "", fmt.Errorf("pdf download returned status %d", resp.StatusCode)
    }

    if p.DefaultPDFPath == "" {
        return "", fmt.Errorf("provider %q has no DefaultPDFPath configured", p.Key)
    }

    if err := writeFileAtomically(p.DefaultPDFPath, resp.Body); err != nil {
        return "", err
    }
    return pdfURL, nil
}
