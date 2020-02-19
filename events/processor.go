package events

import (
	"encoding/json"
	"log"

	"github.com/anothrnick/machinable/dsi/models"
	"github.com/go-redis/redis"
)

const (
	// WebhookQueue is the queue name for web hook callbacks
	WebhookQueue = "hook_queue"
)

// Event defines the event(s) to be processed
type Event struct {
	Project   *models.ProjectDetail `json:"project"`
	Entity    string                `json:"entity"` // resource, json
	EntityKey string                `json:"entity_key"`
	EntityID  string                `json:"entity_id"`
	Action    string                `json:"action"` // create, edit, delete
	Keys      []string              `json:"keys"`
	Payload   []byte                `json:"payload"`
}

// HookEvent describes a single web hook event
type HookEvent struct {
	Hook      *models.WebHook `json:"hook"`
	EntityKey string          `json:"entity_key"`
	Payload   interface{}     `json:"payload"`
}

// Processor process and emits events for web hooks and websockets
type Processor struct {
	cache redis.UniversalClient
}

// NewProcessor creates and returns a new instance of `Processor` with the given redis client
func NewProcessor(cache redis.UniversalClient) *Processor {
	return &Processor{
		cache: cache,
	}
}

// PushEvent processes and emits an event
func (p *Processor) PushEvent(e *Event) error {
	hooks := e.Project.Hooks
	for _, hook := range hooks {
		// emit event to redis for the event action
		if hook.HookEvent == e.Action &&
			hook.Entity == e.Entity &&
			hook.EntityID == e.EntityID &&
			hook.IsEnabled == true {
			hookEvent := &HookEvent{}

			if hook.Entity == "json" {
				var payload interface{}
				json.Unmarshal(e.Payload, &payload)

				container := map[string]interface{}{
					"data": payload,
					"keys": e.Keys,
				}

				hookEvent.Hook = hook
				hookEvent.Payload = container
			} else {
				var payload interface{}
				json.Unmarshal(e.Payload, &payload)
				hookEvent.Hook = hook
				hookEvent.Payload = payload
			}
			hookEvent.EntityKey = e.EntityKey

			b, merr := json.Marshal(hookEvent)
			if merr != nil {
				log.Println(merr)
				continue
			}
			if err := p.cache.RPush(WebhookQueue, b).Err(); err != nil {
				log.Println(err)
			}
		}
	}
	return nil
}
