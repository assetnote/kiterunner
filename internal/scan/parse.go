package scan

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/assetnote/kiterunner/pkg/http"
	"github.com/assetnote/kiterunner/pkg/log"
)

type ErrInvalidProtocol struct {
	Protocol string
	URL      string
}

func (e *ErrInvalidProtocol) Error() string {
	return fmt.Sprintf("Invalid protocol found: %s (%s)", e.Protocol, e.URL)
}

type FileLen struct {
	Filename  string
	MaxLength int
}

func ParseFileWithLen(in string) (FileLen, error) {
	const (
		sep = ":"
	)
	if !strings.Contains(in, sep) {
		return FileLen{
			Filename: in,
		}, nil
	}

	split := strings.SplitN(in, sep, -1)
	before := strings.Join(split[:len(split)-1], sep)
	count := split[len(split)-1]

	val, err := strconv.Atoi(count)
	if err != nil {
		return FileLen{Filename: in}, fmt.Errorf("failed to parse max length. Integer expected: %w", err)
	}

	return FileLen{
		Filename:  before,
		MaxLength: val,
	}, nil
}

// ParseInput will attempt to extract all targets from a given input
// We will attempt to find a file matching your provided <input>, and otherwise
// attempt to parse it as a URI.
// If protocol is missing, then we will assume from the port.
// If the port is missing, then we will try both http:80 and https:443
// "-" should not be passed to this, as we want to parse stdin asynchronously.
func ParseInput(in string) ([]*http.Target, error) {
	ret, err := ParseFile(in)
	if errors.Is(err, os.ErrNotExist) {
		log.Debug().Str("file", in).Msg("file not found. treating as uri")
		return ParseDomain(in)
	}
	return ret, err
}

// ParseFile will perform a ParseDomain on all lines in a file
func ParseFile(filename string) ([]*http.Target, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	defer file.Close()

	ret := make([]*http.Target, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		r, err := ParseDomain(scanner.Text())
		if err != nil {
			return nil, fmt.Errorf("failed to parse domain: %w", err)
		}
		ret = append(ret, r...)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan: %w", err)
	}
	return ret, nil
}

// ParseStdin will return a channel that will publish chunks of targets every second (if there are any targets)
// This attempts to optimise against pipes that slowly write out the targets, allowing us to asynchronously to start
// processing targets without waiting for all the input
func ParseStdin(ctx context.Context) (chan []*http.Target, error) {
	ret := make(chan []*http.Target, 10)
	child, cancel := context.WithCancel(ctx)

	var (
		input   []string
		inputmu sync.Mutex
	)

	// scan async in a thread. We append across a thread
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		defer cancel()
		for scanner.Scan() {
			inputmu.Lock()
			input = append(input, scanner.Text())
			inputmu.Unlock()

			select {
			case <-child.Done():
				// log.Trace().Err(child.Err()).Str("goroutine", "stdin scanner").Msg("received context cancellation")
				return
			default:
			}
		}

		if err := scanner.Err(); err != nil {
			log.Error().Err(err).Msg("failed to scan")
		}
		return
	}()

	go func() {
		tick := time.Tick(1 * time.Second)
		var (
			quit = false
		)
		for {
			select {
			case <-child.Done():
				if !quit {
					// log.Trace().Err(child.Err()).Str("goroutine", "stdin tick worker").Msg("received context cancellation")
					quit = true
				}
				// don't immedaitely exit, wait for next tick so we drain the channel then exit

			case <-tick:
				// grab all our domains from the last batch
				inputmu.Lock()
				tmp := make([]string, len(input))
				copy(tmp, input)
				input = input[0:0]
				inputmu.Unlock()

				// create our targets
				send := make([]*http.Target, 0)
				for _, v := range tmp {
					r, err := ParseDomain(v)
					if err != nil {
						log.Error().Err(err).Msg("failed to parse domain")
						continue
					}
					send = append(send, r...)
				}

				// send them off
				ret <- send

				if quit {
					close(ret)
					return
				}
			}
		}
	}()
	return ret, nil
}

// ParseDomain will attempt to determine the target based off the input
// The only support protocols are http, https
// If protocol is missing, then we will assume from the port.
// If the port is missing, then we will try both http:80 and https:443
// we use net/url to parse the URL
func ParseDomain(domain string) ([]*http.Target, error) {
	ret := make([]*http.Target, 0)

	// tmp is used to store what we parse, we then later determine whether to duplicate based on the port/protocol
	var (
		guessProto bool
	)
	// if the string has no protocol, then we should guess the protocol. Temporarily prepend the proto for now
	if !strings.Contains(domain, "://") {
		domain = "http://" + domain
		guessProto = true
	}

	// parse the port if there is one
	parsed, err := url.Parse(domain)
	if err != nil {
		return nil, err
	}

	if parsed.Port() == "" && guessProto {
		// if both the port and proto are missing, we guess both https:443 and http:80
		t := http.AcquireTarget()
		t.Hostname = parsed.Hostname()
		t.Port = 80
		t.IsTLS = false
		t.BasePath = parsed.Path
		ret = append(ret, t)

		t = http.AcquireTarget()
		t.Hostname = parsed.Hostname()
		t.Port = 443
		t.IsTLS = true
		t.BasePath = parsed.Path
		ret = append(ret, t)

	} else if guessProto && parsed.Port() != "" {
		// port but no proto
		tmp := http.AcquireTarget()
		tmp.Hostname = parsed.Hostname()
		tmp.BasePath = parsed.Path

		tmp.Port, err = strconv.Atoi(parsed.Port())
		if err != nil {
			return nil, fmt.Errorf("unable to parse port: %w", err)
		}
		switch tmp.Port {
		case 443, 8443:
			tmp.IsTLS = true
		}
		ret = append(ret, tmp)
	} else if !guessProto && parsed.Port() == "" {
		// proto but no port
		tmp := http.AcquireTarget()
		tmp.Hostname = parsed.Hostname()
		tmp.BasePath = parsed.Path

		switch parsed.Scheme {
		case "http":
			tmp.IsTLS = false
			tmp.Port = 80
		case "https":
			tmp.IsTLS = true
			tmp.Port = 443
		default:
			return nil, &ErrInvalidProtocol{Protocol: parsed.Scheme, URL: domain}
		}
		ret = append(ret, tmp)
	} else {
		// We have everything
		tmp := http.AcquireTarget()
		tmp.Hostname = parsed.Hostname()
		tmp.BasePath = parsed.Path

		tmp.Port, err = strconv.Atoi(parsed.Port())
		if err != nil {
			return nil, fmt.Errorf("unable to parse port: %w", err)
		}

		switch parsed.Scheme {
		case "http":
			tmp.IsTLS = false
		case "https":
			tmp.IsTLS = true
		default:
			return nil, &ErrInvalidProtocol{Protocol: parsed.Scheme, URL: domain}
		}
		ret = append(ret, tmp)
	}

	return ret, nil
}
