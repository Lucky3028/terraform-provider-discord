package discord

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Config struct {
	Token    string
	ClientID string
	Secret   string
}

type Context struct {
	Session *discordgo.Session
	//Client  *disgord.Client
	Config *Config
}

// LimitedRoundTripper This type implements the http.RoundTripper interface
type LimitedRoundTripper struct {
	Proxied http.RoundTripper
}

func (lrt LimitedRoundTripper) RoundTrip(req *http.Request) (res *http.Response, e error) {
	// Do "before sending requests" actions here.
	fmt.Printf("Sending request to %v\n", req.URL)

	// Send the request, get the response (or the error)
	res, e = lrt.Proxied.RoundTrip(req)

	if res != nil && res.StatusCode == 429 {
		retryAfter := res.Header.Get("X-RateLimit-Reset-After")
		if retryAfter == "" {
			retryAfter = res.Header.Get("Retry-After")
		}

		dur, _ := strconv.Atoi(retryAfter)
		time.Sleep(time.Duration(dur) * time.Millisecond)

		return lrt.RoundTrip(req)
	}

	return
}

func (c *Config) Client() (*Context, error) {
	//httpClient := &http.Client{Transport: LimitedRoundTripper{http.DefaultTransport}}
	//client := disgord.New(disgord.Config{
	//	BotToken:   c.Token,
	//	HTTPClient: httpClient,
	//})
	session, err := discordgo.New(c.Token)
	if err != nil {
		return nil, err
	}

	//return &Context{Client: client, Config: c, Session: session}, nil
	return &Context{Config: c, Session: session}, nil
}
