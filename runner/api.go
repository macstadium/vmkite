package runner

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/macstadium/vmkite/buildkite"
)

type apiHookEvent struct {
	JobID     string
	Event     string
	Timestamp time.Time
}

type api struct {
	sync.Mutex
	net.Listener

	subscribers map[string]chan apiHookEvent
	authTokens  map[string]string
	secret      string
}

func newApiListener(listenOn string, tokenSecret string) (*api, error) {
	if listenOn == "" {
		addr, err := getLocalIP()
		if err != nil {
			return nil, err
		}
		listenOn = addr + ":0"
	}

	if tokenSecret == "" {
		tokenSecret = fmt.Sprintf("%x", time.Now().UnixNano())
	}

	l, err := net.Listen("tcp", listenOn)
	if err != nil {
		return nil, err
	}

	server := &api{
		Listener:    l,
		subscribers: map[string]chan apiHookEvent{},
		authTokens:  map[string]string{},
		secret:      tokenSecret,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		json.NewEncoder(w).Encode("OK")
	})

	mux.HandleFunc("/notify/hook/", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(w, "Only POST is Allowed", http.StatusBadRequest)
		}
		server.authenticate(server.handleNotifyHook).ServeHTTP(w, req)
	})

	go func() {
		debugf("API server listening on %s", l.Addr().String())
		fmt.Println(http.Serve(l, mux))
		os.Exit(1)
	}()

	return server, nil
}

func (a *api) Subscribe(job buildkite.VmkiteJob) (string, chan apiHookEvent, error) {
	events := make(chan apiHookEvent)
	data := make([]byte, 10)

	_, err := rand.Read(data)
	if err != nil {
		return "", nil, err
	}

	key := []byte(a.secret)
	h := hmac.New(sha256.New, key)
	h.Write([]byte(job.ID))
	token := base64.StdEncoding.EncodeToString(h.Sum(nil))

	a.Lock()
	defer a.Unlock()

	if _, ok := a.subscribers[job.ID]; ok {
		return "", nil, fmt.Errorf("A subscriber for %v already exists", job.ID)
	}

	a.authTokens[token] = job.ID
	a.subscribers[job.ID] = events

	return token, events, nil
}

func (a *api) Release(job buildkite.VmkiteJob) {
	if events, ok := a.subscribers[job.ID]; ok {
		debugf("Releasing subscriber for %v", job.ID)
		close(events)
	}
}

func (a *api) handleNotifyHook(w http.ResponseWriter, req *http.Request) {
	jobID := req.Context().Value("JobID")
	if jobID == nil {
		http.Error(w, "Unknown job id", http.StatusBadRequest)
		return
	}

	hook := path.Base(req.URL.Path)
	debugf("job %s reported hook %s", jobID, hook)

	a.Lock()
	defer a.Unlock()

	events, ok := a.subscribers[jobID.(string)]
	if !ok {
		http.Error(w, "Unknown hook", http.StatusNotFound)
		return
	}

	events <- apiHookEvent{
		JobID:     jobID.(string),
		Timestamp: time.Now(),
		Event:     hook,
	}

	json.NewEncoder(w).Encode("OK")
}

// authenticate provides Authentication middleware for handlers
func (a *api) authenticate(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var token string

		// Get token from the Authorization header
		// format: Authorization: Bearer
		tokens, ok := r.Header["Authorization"]
		if ok && len(tokens) >= 1 {
			token = tokens[0]
			token = strings.TrimPrefix(token, "Bearer ")
		}

		// Check if we have the token in our auth table
		jobID, ok := a.authTokens[token]
		if !ok {
			debugf("Got incorrect auth token of %s", token)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), "JobID", jobID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", err
}
