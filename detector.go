package botdetector

import (
	"net/http"
	"regexp"

	lru "github.com/hashicorp/golang-lru"
)

// Config defines the behaviour of the detector
type Config struct {
	Denylist  []string `json:"deny"`
	Allowlist []string `json:"allow"`
	Patterns  []string `json:"patterns"`
	CacheSize int      `json:"cache_size"`
}

// DetectorFunc is a func that chek if a request was made by a bot
type DetectorFunc func(r *http.Request) bool

// New returns a detector function with or without LRU cache depending on the params
func New(cfg Config) (DetectorFunc, error) {
	if cfg.CacheSize == 0 {
		d, err := NewDetector(cfg)
		return d.IsBot, err
	}

	d, err := NewLRU(cfg)
	return d.IsBot, err
}

// NewDetector creates a Detector
func NewDetector(cfg Config) (*Detector, error) {
	deny := make(map[string]struct{}, len(cfg.Denylist))
	for _, e := range cfg.Denylist {
		deny[e] = struct{}{}
	}
	allow := make(map[string]struct{}, len(cfg.Allowlist))
	for _, e := range cfg.Allowlist {
		allow[e] = struct{}{}
	}
	patterns := make([]*regexp.Regexp, len(cfg.Patterns))
	for i, p := range cfg.Patterns {
		rp, err := regexp.Compile(p)
		if err != nil {
			return nil, err
		}
		patterns[i] = rp
	}
	return &Detector{
		deny:     deny,
		allow:    allow,
		patterns: patterns,
	}, nil
}

// Detector is a struct able to detect bot-made requests
type Detector struct {
	deny     map[string]struct{}
	allow    map[string]struct{}
	patterns []*regexp.Regexp
}

// IsBot returns true if the request was made by a bot
func (d *Detector) IsBot(r *http.Request) bool {
	userAgent := r.Header.Get("User-Agent")

	if userAgent == "" {
		return false
	}
	if _, ok := d.allow[userAgent]; ok {
		return false
	}
	if _, ok := d.deny[userAgent]; ok {
		return true
	}
	for _, p := range d.patterns {
		if p.MatchString(userAgent) {
			return true
		}
	}
	return false
}

// NewLRU creates a new LRUDetector
func NewLRU(cfg Config) (*LRUDetector, error) {
	d, err := NewDetector(cfg)
	if err != nil {
		return nil, err
	}

	cache, err := lru.New(cfg.CacheSize)
	if err != nil {
		return nil, err
	}

	return &LRUDetector{
		detectorFunc: d.IsBot,
		cache:        cache,
	}, nil
}

// LRUDetector is a struct able to detect bot-made requests and cache the results
// for future reutilization
type LRUDetector struct {
	detectorFunc DetectorFunc
	cache        *lru.Cache
}

// IsBot returns true if the request was made by a bot
func (d *LRUDetector) IsBot(r *http.Request) bool {
	userAgent := r.Header.Get("User-Agent")
	cached, ok := d.cache.Get(userAgent)
	if ok {
		return cached.(bool)
	}

	res := d.detectorFunc(r)
	d.cache.Add(userAgent, res)

	return res
}
