package pubnub

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pubnub/go/pnerr"
)

const grantPath = "/v1/auth/grant/sub-key/%s"

var emptyGrantResponse *GrantResponse

type grantBuilder struct {
	opts *grantOpts
}

func newGrantBuilder(pubnub *PubNub) *grantBuilder {
	builder := grantBuilder{
		opts: &grantOpts{
			pubnub: pubnub,
		},
	}

	return &builder
}

func newGrantBuilderWithContext(pubnub *PubNub, context Context) *grantBuilder {
	builder := grantBuilder{
		opts: &grantOpts{
			pubnub: pubnub,
			ctx:    context,
		},
	}

	return &builder
}

func (b *grantBuilder) Read(read bool) *grantBuilder {
	b.opts.Read = read

	return b
}

func (b *grantBuilder) Write(write bool) *grantBuilder {
	b.opts.Write = write

	return b
}

func (b *grantBuilder) Manage(manage bool) *grantBuilder {
	b.opts.Manage = manage

	return b
}

// TTL in minutes for which granted permissions are valid.
//
// Min: 1
// Max: 525600
// Default: 1440
//
// Setting value to 0 will apply the grant indefinitely (forever grant).
func (b *grantBuilder) TTL(ttl int) *grantBuilder {
	b.opts.TTL = ttl
	b.opts.setTTL = true

	return b
}

// AuthKeys sets the AuthKeys for the Grant request.
func (b *grantBuilder) AuthKeys(authKeys []string) *grantBuilder {
	b.opts.AuthKeys = authKeys

	return b
}

// Channels sets the Channels for the Grant request.
func (b *grantBuilder) Channels(channels []string) *grantBuilder {
	b.opts.Channels = channels

	return b
}

// ChannelGroups sets the ChannelGroups for the Grant request.
func (b *grantBuilder) ChannelGroups(groups []string) *grantBuilder {
	b.opts.ChannelGroups = groups

	return b
}

// Execute runs the Grant request.
func (b *grantBuilder) Execute() (*GrantResponse, StatusResponse, error) {
	rawJSON, status, err := executeRequest(b.opts)
	if err != nil {
		return emptyGrantResponse, status, err
	}

	return newGrantResponse(rawJSON, status)
}

type grantOpts struct {
	pubnub *PubNub
	ctx    Context

	AuthKeys      []string
	Channels      []string
	ChannelGroups []string

	// Stringified permissions
	// Setting 'true' or 'false' will apply permissions to level
	Read   bool
	Write  bool
	Manage bool

	// Max: 525600
	// Min: 1
	// Default: 1440
	// Setting 0 will apply the grant indefinitely
	TTL int

	// nil hacks
	setTTL bool
}

func (o *grantOpts) config() Config {
	return *o.pubnub.Config
}

func (o *grantOpts) client() *http.Client {
	return o.pubnub.GetClient()
}

func (o *grantOpts) context() Context {
	return o.ctx
}

func (o *grantOpts) validate() error {
	if o.config().PublishKey == "" {
		return newValidationError(o, StrMissingPubKey)
	}

	if o.config().SubscribeKey == "" {
		return newValidationError(o, StrMissingSubKey)
	}

	if o.config().SecretKey == "" {
		return newValidationError(o, StrMissingSecretKey)
	}

	return nil
}

func (o *grantOpts) buildPath() (string, error) {
	return fmt.Sprintf(grantPath, o.pubnub.Config.SubscribeKey), nil
}

func (o *grantOpts) buildQuery() (*url.Values, error) {
	q := defaultQuery(o.pubnub.Config.UUID, o.pubnub.telemetryManager)

	if o.Read {
		q.Set("r", "1")
	} else {
		q.Set("r", "0")
	}

	if o.Write {
		q.Set("w", "1")
	} else {
		q.Set("w", "0")
	}

	if o.Manage {
		q.Set("m", "1")
	} else {
		q.Set("m", "0")
	}

	if len(o.AuthKeys) > 0 {
		q.Set("auth", strings.Join(o.AuthKeys, ","))
	}

	if len(o.Channels) > 0 {
		q.Set("channel", strings.Join(o.Channels, ","))
	}

	if len(o.ChannelGroups) > 0 {
		q.Set("channel-group", strings.Join(o.ChannelGroups, ","))
	}

	if o.setTTL {
		if o.TTL >= -1 {
			q.Set("ttl", fmt.Sprintf("%d", o.TTL))
		}
	}

	timestamp := time.Now().Unix()
	q.Set("timestamp", strconv.Itoa(int(timestamp)))

	return q, nil
}

func (o *grantOpts) buildBody() ([]byte, error) {
	return []byte{}, nil
}

func (o *grantOpts) httpMethod() string {
	return "GET"
}

func (o *grantOpts) isAuthRequired() bool {
	return true
}

func (o *grantOpts) requestTimeout() int {
	return o.pubnub.Config.NonSubscribeRequestTimeout
}

func (o *grantOpts) connectTimeout() int {
	return o.pubnub.Config.ConnectTimeout
}

func (o *grantOpts) operationType() OperationType {
	return PNAccessManagerGrant
}

func (o *grantOpts) telemetryManager() *TelemetryManager {
	return o.pubnub.telemetryManager
}

// GrantResponse is the struct returned when the Execute function of Grant is called.
type GrantResponse struct {
	Level        string
	SubscribeKey string

	TTL int

	Channels      map[string]*PNPAMEntityData
	ChannelGroups map[string]*PNPAMEntityData

	ReadEnabled   bool
	WriteEnabled  bool
	ManageEnabled bool
}

// PNPAMEntityData is the struct containing the access details of the channels.
type PNPAMEntityData struct {
	Name          string
	AuthKeys      map[string]*PNAccessManagerKeyData
	ReadEnabled   bool
	WriteEnabled  bool
	ManageEnabled bool
	TTL           int
}

// PNAccessManagerKeyData is the struct containing the access details of the channel groups.
type PNAccessManagerKeyData struct {
	ReadEnabled   bool
	WriteEnabled  bool
	ManageEnabled bool
	TTL           int
}

func newGrantResponse(jsonBytes []byte, status StatusResponse) (
	*GrantResponse, StatusResponse, error) {
	resp := &GrantResponse{}
	var value interface{}

	err := json.Unmarshal(jsonBytes, &value)
	if err != nil {
		e := pnerr.NewResponseParsingError("Error unmarshalling response",
			ioutil.NopCloser(bytes.NewBufferString(string(jsonBytes))), err)

		return emptyGrantResponse, status, e
	}

	constructedChannels := make(map[string]*PNPAMEntityData)
	constructedGroups := make(map[string]*PNPAMEntityData)

	grantData, _ := value.(map[string]interface{})
	payload := grantData["payload"]
	parsedPayload := payload.(map[string]interface{})
	auths, _ := parsedPayload["auths"].(map[string]interface{})
	ttl, _ := parsedPayload["ttl"].(float64)

	if val, ok := parsedPayload["channel"]; ok {
		channelName := val.(string)
		auths := make(map[string]*PNAccessManagerKeyData)
		channelMap, _ := parsedPayload["auths"].(map[string]interface{})
		entityData := &PNPAMEntityData{
			Name: channelName,
		}

		for key, value := range channelMap {
			valueMap := value.(map[string]interface{})
			keyData := &PNAccessManagerKeyData{}

			if val, ok := valueMap["r"]; ok {
				parsedValue, _ := val.(float64)
				if parsedValue == float64(1) {
					keyData.ReadEnabled = true
				} else {
					keyData.ReadEnabled = false
				}
			}

			if val, ok := valueMap["w"]; ok {
				parsedValue, _ := val.(float64)
				if parsedValue == float64(1) {
					keyData.WriteEnabled = true
				} else {
					keyData.WriteEnabled = false
				}
			}

			if val, ok := valueMap["m"]; ok {
				parsedValue, _ := val.(float64)
				if parsedValue == float64(1) {
					keyData.ManageEnabled = true
				} else {
					keyData.ManageEnabled = false
				}
			}

			auths[key] = keyData
		}

		entityData.AuthKeys = auths
		entityData.TTL = int(ttl)
		constructedChannels[channelName] = entityData
	}

	if val, ok := parsedPayload["channel-groups"]; ok {
		groupName, _ := val.(string)
		constructedAuthKey := make(map[string]*PNAccessManagerKeyData)
		entityData := PNPAMEntityData{
			Name: groupName,
		}

		if _, ok := val.(string); ok {
			for authKeyName, value := range auths {
				auth, _ := value.(map[string]interface{})

				managerKeyData := &PNAccessManagerKeyData{}

				if val, ok := auth["r"]; ok {
					parsedValue, _ := val.(float64)
					if parsedValue == float64(1) {
						managerKeyData.ReadEnabled = true
					} else {
						managerKeyData.ReadEnabled = false
					}
				}

				if val, ok := auth["w"]; ok {
					parsedValue, _ := val.(float64)
					if parsedValue == float64(1) {
						managerKeyData.WriteEnabled = true
					} else {
						managerKeyData.WriteEnabled = false
					}
				}

				if val, ok := auth["m"]; ok {
					parsedValue, _ := val.(float64)
					if parsedValue == float64(1) {
						managerKeyData.ManageEnabled = true
					} else {
						managerKeyData.ManageEnabled = false
					}
				}

				if val, ok := auth["ttl"]; ok {
					parsedVal, _ := val.(int)
					entityData.TTL = parsedVal
				}

				constructedAuthKey[authKeyName] = managerKeyData
			}

			entityData.AuthKeys = constructedAuthKey
			constructedGroups[groupName] = &entityData
		}

		if groupMap, ok := val.(map[string]interface{}); ok {
			groupName, _ := val.(string)
			constructedAuthKey := make(map[string]*PNAccessManagerKeyData)
			entityData := PNPAMEntityData{
				Name: groupName,
			}

			for groupName, value := range groupMap {
				valueMap := value.(map[string]interface{})

				if keys, ok := valueMap["auths"]; ok {
					parsedKeys, _ := keys.(map[string]interface{})
					keyData := &PNAccessManagerKeyData{}

					for keyName, value := range parsedKeys {
						valueMap, _ := value.(map[string]interface{})

						if val, ok := valueMap["r"]; ok {
							parsedValue, _ := val.(float64)
							if parsedValue == float64(1) {
								keyData.ReadEnabled = true
							} else {
								keyData.ReadEnabled = false
							}
						}

						if val, ok := valueMap["w"]; ok {
							parsedValue, _ := val.(float64)
							if parsedValue == float64(1) {
								keyData.WriteEnabled = true
							} else {
								keyData.WriteEnabled = false
							}
						}

						if val, ok := valueMap["m"]; ok {
							parsedValue, _ := val.(float64)
							if parsedValue == float64(1) {
								keyData.ManageEnabled = true
							} else {
								keyData.ManageEnabled = false
							}
						}

						constructedAuthKey[keyName] = keyData
					}
				}

				if val, ok := valueMap["r"]; ok {
					parsedValue, _ := val.(float64)
					if parsedValue == float64(1) {
						entityData.ReadEnabled = true
					} else {
						entityData.ReadEnabled = false
					}
				}

				if val, ok := valueMap["w"]; ok {
					parsedValue, _ := val.(float64)
					if parsedValue == float64(1) {
						entityData.WriteEnabled = true
					} else {
						entityData.WriteEnabled = false
					}
				}

				if val, ok := valueMap["m"]; ok {
					parsedValue, _ := val.(float64)
					if parsedValue == float64(1) {
						entityData.ManageEnabled = true
					} else {
						entityData.ManageEnabled = false
					}
				}

				if val, ok := parsedPayload["ttl"]; ok {
					parsedVal, _ := val.(float64)
					entityData.TTL = int(parsedVal)
				}

				entityData.AuthKeys = constructedAuthKey
				constructedGroups[groupName] = &entityData
			}
		}
	}

	if val, ok := parsedPayload["channels"]; ok {
		channelMap, _ := val.(map[string]interface{})

		for channelName, value := range channelMap {
			constructedChannels[channelName] = fetchChannel(channelName,
				value, parsedPayload)
		}
	}

	level, _ := parsedPayload["level"].(string)
	subKey, _ := parsedPayload["subscribe_key"].(string)

	resp.Level = level
	resp.SubscribeKey = subKey
	resp.Channels = constructedChannels
	resp.ChannelGroups = constructedGroups

	if r, ok := parsedPayload["r"]; ok {
		parsedValue, _ := r.(float64)
		if parsedValue == float64(1) {
			resp.ReadEnabled = true
		} else {
			resp.ReadEnabled = false
		}
	}

	if r, ok := parsedPayload["w"]; ok {
		parsedValue, _ := r.(float64)
		if parsedValue == float64(1) {
			resp.WriteEnabled = true
		} else {
			resp.WriteEnabled = false
		}
	}

	if r, ok := parsedPayload["m"]; ok {
		parsedValue, _ := r.(float64)
		if parsedValue == float64(1) {
			resp.ManageEnabled = true
		} else {
			resp.ManageEnabled = false
		}
	}

	if r, ok := parsedPayload["ttl"]; ok {
		parsedValue, _ := r.(float64)
		resp.TTL = int(parsedValue)
	}

	return resp, status, nil
}

func fetchChannel(channelName string,
	value interface{}, parsedPayload map[string]interface{}) *PNPAMEntityData {

	auths := make(map[string]*PNAccessManagerKeyData)
	entityData := &PNPAMEntityData{
		Name: channelName,
	}

	valueMap, _ := value.(map[string]interface{})

	if val, ok := valueMap["auths"]; ok {
		parsedValue := val.(map[string]interface{})

		for key, value := range parsedValue {
			valueMap := value.(map[string]interface{})
			keyData := &PNAccessManagerKeyData{}

			if val, ok := valueMap["r"]; ok {
				parsedValue, _ := val.(float64)
				if parsedValue == float64(1) {
					keyData.ReadEnabled = true
				} else {
					keyData.ReadEnabled = false
				}
			}

			if val, ok := valueMap["w"]; ok {
				parsedValue, _ := val.(float64)
				if parsedValue == float64(1) {
					keyData.WriteEnabled = true
				} else {
					keyData.WriteEnabled = false
				}
			}

			if val, ok := valueMap["m"]; ok {
				parsedValue, _ := val.(float64)
				if parsedValue == float64(1) {
					keyData.ManageEnabled = true
				} else {
					keyData.ManageEnabled = false
				}
			}

			auths[key] = keyData
		}
	}

	if val, ok := valueMap["r"]; ok {
		parsedValue, _ := val.(float64)
		if parsedValue == float64(1) {
			entityData.ReadEnabled = true
		} else {
			entityData.ReadEnabled = false
		}
	}

	if val, ok := valueMap["w"]; ok {
		parsedValue, _ := val.(float64)
		if parsedValue == float64(1) {
			entityData.WriteEnabled = true
		} else {
			entityData.WriteEnabled = false
		}
	}

	if val, ok := valueMap["m"]; ok {
		parsedValue, _ := val.(float64)
		if parsedValue == float64(1) {
			entityData.ManageEnabled = true
		} else {
			entityData.ManageEnabled = false
		}
	}

	if val, ok := parsedPayload["ttl"]; ok {
		parsedVal, _ := val.(float64)
		entityData.TTL = int(parsedVal)
	}

	entityData.AuthKeys = auths

	return entityData
}