package e2e

import (
	//"encoding/json"
	"fmt"
	//"log"
	"math/rand"
	//"os"
	//"reflect"
	"sync"
	"testing"
	"time"

	pubnub "github.com/pubnub/go"
	"github.com/pubnub/go/tests/stubs"
	"github.com/stretchr/testify/assert"
)

/////////////////////////////
/////////////////////////////
// Structure
// - Channel Subscription
// - Groups Subscription
// - Misc
/////////////////////////////
/////////////////////////////

/////////////////////////////
// Channel Subscription
/////////////////////////////

func TestSubscribeUnsubscribe(t *testing.T) {
	assert := assert.New(t)
	doneSubscribe := make(chan bool)
	doneUnsubscribe := make(chan bool)
	errChan := make(chan string)
	ch := randomized("sub-u-ch")

	interceptor := stubs.NewInterceptor()
	interceptor.AddStub(&stubs.Stub{
		Method:             "GET",
		Path:               fmt.Sprintf("/v2/subscribe/sub-c-e41d50d4-43ce-11e8-a433-9e6b275e7b64/%s/0", ch),
		Query:              "heartbeat=300",
		ResponseBody:       `{"t":{"t":"15079041051785708","r":12},"m":[]}`,
		IgnoreQueryKeys:    []string{"pnsdk", "uuid", "tt"},
		ResponseStatusCode: 200,
	})

	pn := pubnub.NewPubNub(configCopy())
	pn.SetSubscribeClient(interceptor.GetClient())

	listener := pubnub.NewListener()

	go func() {
		for {
			select {
			case status := <-listener.Status:
				switch status.Category {
				case pubnub.PNConnectedCategory:
					doneSubscribe <- true
				case pubnub.PNDisconnectedCategory:
					doneUnsubscribe <- true
					return
				}
			case <-listener.Message:
				errChan <- "Got message while awaiting for a status event"
				return
			case <-listener.Presence:
				errChan <- "Got presence while awaiting for a status event"
				return
			}
		}
	}()

	pn.AddListener(listener)

	pn.Subscribe().Channels([]string{ch}).Execute()

	select {
	case <-doneSubscribe:
	case err := <-errChan:
		assert.Fail(err)
	}

	pn.Unsubscribe().
		Channels([]string{ch}).
		Execute()

	select {
	case <-doneUnsubscribe:
	case err := <-errChan:
		assert.Fail(err)
	}

	assert.Zero(len(pn.GetSubscribedChannels()))
	assert.Zero(len(pn.GetSubscribedGroups()))
}

func GenRandom() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

func TestSubscribePublishUnsubscribeString(t *testing.T) {
	assert := assert.New(t)
	pubMessage := make(chan interface{})
	s := "hey"

	go SubscribePublishUnsubscribeMultiCommon(t, s, "", pubMessage, false, false)
	m := <-pubMessage
	msg := m.(string)
	assert.Equal(s, msg)
}

func TestSubscribePublishUnsubscribeStringEnc(t *testing.T) {
	assert := assert.New(t)
	pubMessage := make(chan interface{})
	s := "yay!"

	go SubscribePublishUnsubscribeMultiCommon(t, s, "enigma", pubMessage, false, false)
	m := <-pubMessage
	msg := m.(string)
	assert.Equal(s, msg)
}

func TestSubscribePublishUnsubscribeInt(t *testing.T) {
	assert := assert.New(t)
	pubMessage := make(chan interface{})
	s := 1

	go SubscribePublishUnsubscribeMultiCommon(t, s, "", pubMessage, false, false)
	m := <-pubMessage
	msg := m.(float64)
	assert.Equal(float64(1), msg)
}

func TestSubscribePublishUnsubscribeIntEnc(t *testing.T) {
	assert := assert.New(t)
	pubMessage := make(chan interface{})
	s := 1

	go SubscribePublishUnsubscribeMultiCommon(t, s, "enigma", pubMessage, false, false)
	m := <-pubMessage
	msg := m.(float64)
	assert.Equal(float64(1), msg)
}

func TestSubscribePublishUnsubscribePNOtherDisable(t *testing.T) {
	assert := assert.New(t)
	pubMessage := make(chan interface{})
	s := map[string]interface{}{
		"id":        2,
		"not_other": "123456",
		"pn_other":  "yay!",
	}

	go SubscribePublishUnsubscribeMultiCommon(t, s, "enigma", pubMessage, true, false)
	m := <-pubMessage
	msg := m.(map[string]interface{})
	assert.Equal("123456", msg["not_other"])
	assert.Equal("yay!", msg["pn_other"])
}

func TestSubscribePublishUnsubscribePNOther(t *testing.T) {
	assert := assert.New(t)
	pubMessage := make(chan interface{})
	s := map[string]interface{}{
		"id":        1,
		"not_other": "12345",
		"pn_other":  "yay!",
	}

	go SubscribePublishUnsubscribeMultiCommon(t, s, "enigma", pubMessage, false, false)
	m := <-pubMessage
	msg := m.(map[string]interface{})
	assert.Equal("12345", msg["not_other"])
	assert.Equal("yay!", msg["pn_other"])

}

func TestSubscribePublishUnsubscribePNOtherComplex(t *testing.T) {
	assert := assert.New(t)
	pubMessage := make(chan interface{})
	s1 := map[string]interface{}{
		"id":        1,
		"not_other": "1234567",
	}
	s := map[string]interface{}{
		"id":        1,
		"not_other": "12345",
		"pn_other":  s1,
	}

	go SubscribePublishUnsubscribeMultiCommon(t, s, "enigma", pubMessage, false, false)
	m := <-pubMessage
	msg := m.(map[string]interface{})
	assert.Equal("12345", msg["not_other"])
	if msgOther, ok := msg["pn_other"].(map[string]interface{}); !ok {
		assert.Fail("!map[string]interface{}")
	} else {
		assert.Equal("1234567", msgOther["not_other"])
	}

}

func TestSubscribePublishUnsubscribeInterfaceWithoutPNOther(t *testing.T) {
	assert := assert.New(t)
	pubMessage := make(chan interface{})
	s := map[string]interface{}{
		"id":        3,
		"not_other": "1234",
		"ss_other":  "yay!",
	}

	go SubscribePublishUnsubscribeMultiCommon(t, s, "", pubMessage, false, false)
	m := <-pubMessage
	msg := m.(map[string]interface{})
	assert.Equal("1234", msg["not_other"])
	assert.Equal("yay!", msg["ss_other"])

}

func TestSubscribePublishUnsubscribeInterfaceWithoutPNOtherEnc(t *testing.T) {
	assert := assert.New(t)
	pubMessage := make(chan interface{})
	s := map[string]interface{}{
		"id":        4,
		"not_other": "123",
		"ss_other":  "yay!",
	}

	go SubscribePublishUnsubscribeMultiCommon(t, s, "enigma", pubMessage, false, false)
	m := <-pubMessage
	msg := m.(map[string]interface{})
	assert.Equal("123", msg["not_other"])
	assert.Equal("yay!", msg["ss_other"])
}

type customStruct struct {
	Foo string
	Bar []int
}

func TestSubscribePublishUnsubscribeInterfaceCustom(t *testing.T) {
	assert := assert.New(t)
	pubMessage := make(chan interface{})
	s := customStruct{
		Foo: "hi!",
		Bar: []int{1, 2, 3, 4, 5},
	}

	go SubscribePublishUnsubscribeMultiCommon(t, s, "", pubMessage, false, false)
	m := <-pubMessage
	//s1 := reflect.ValueOf(m)
	//fmt.Println("s:::", s1, s1.Type())
	if msg, ok := m.(map[string]interface{}); !ok {
		//fmt.Println(msg)
		assert.Fail("not map")
	} else {
		//fmt.Println(msg)
		//byt := []byte(message.Message)
		//fmt.Println(message.Message.(string))
		//err := json.Unmarshal(byt, &msg)
		//assert.Nil(err)
		assert.Equal("hi!", msg["Foo"])
		//assert.Equal("1", msg["Bar"].(map[string]interface{})[0])
		//assert.Equal("\"yay!\"", msg["pn_other"])
	}
}

func TestSubscribePublishUnsubscribeInterfaceWithoutCustomEnc(t *testing.T) {
	assert := assert.New(t)
	pubMessage := make(chan interface{})
	s := customStruct{
		Foo: "hi!",
		Bar: []int{1, 2, 3, 4, 5},
	}

	go SubscribePublishUnsubscribeMultiCommon(t, s, "enigma", pubMessage, false, false)
	m := <-pubMessage
	//s1 := reflect.ValueOf(m)
	//fmt.Println("s:::", s1, s1.Type())
	if msg, ok := m.(map[string]interface{}); !ok {
		//fmt.Println(msg)
		assert.Fail("not map")
	} else {
		//fmt.Println(msg)
		//byt := []byte(message.Message)
		//fmt.Println(message.Message.(string))
		//err := json.Unmarshal(byt, &msg)
		//assert.Nil(err)
		assert.Equal("hi!", msg["Foo"])
		//assert.Equal("1", msg["Bar"].(map[string]interface{})[0])
		//assert.Equal("\"yay!\"", msg["pn_other"])
	}
}

func TestSubscribePublishUnsubscribeStringPost(t *testing.T) {
	assert := assert.New(t)
	pubMessage := make(chan interface{})
	s := "hey"

	go SubscribePublishUnsubscribeMultiCommon(t, s, "", pubMessage, false, true)
	m := <-pubMessage
	msg := m.(string)
	assert.Equal(s, msg)
}

func TestSubscribePublishUnsubscribeStringEncPost(t *testing.T) {
	assert := assert.New(t)
	pubMessage := make(chan interface{})
	s := "hey"

	go SubscribePublishUnsubscribeMultiCommon(t, s, "enigma", pubMessage, false, true)
	m := <-pubMessage
	msg := m.(string)
	assert.Equal(s, msg)
}

func TestSubscribePublishUnsubscribeIntPost(t *testing.T) {
	assert := assert.New(t)
	pubMessage := make(chan interface{})
	s := 1

	go SubscribePublishUnsubscribeMultiCommon(t, s, "", pubMessage, false, true)
	m := <-pubMessage
	msg := m.(float64)
	assert.Equal(float64(1), msg)
}

func TestSubscribePublishUnsubscribeIntEncPost(t *testing.T) {
	assert := assert.New(t)
	pubMessage := make(chan interface{})
	s := 1

	go SubscribePublishUnsubscribeMultiCommon(t, s, "enigma", pubMessage, false, true)
	m := <-pubMessage
	msg := m.(float64)
	assert.Equal(float64(1), msg)
}

func TestSubscribePublishUnsubscribePNOtherDisablePost(t *testing.T) {
	assert := assert.New(t)
	pubMessage := make(chan interface{})
	s := map[string]interface{}{
		"id":        2,
		"not_other": "123456",
		"pn_other":  "yay!",
	}

	go SubscribePublishUnsubscribeMultiCommon(t, s, "enigma", pubMessage, true, true)
	m := <-pubMessage
	msg := m.(map[string]interface{})
	assert.Equal("123456", msg["not_other"])
	assert.Equal("yay!", msg["pn_other"])
}

func TestSubscribePublishUnsubscribePNOtherPost(t *testing.T) {
	assert := assert.New(t)
	pubMessage := make(chan interface{})
	s := map[string]interface{}{
		"id":        1,
		"not_other": "12345",
		"pn_other":  "yay!",
	}

	go SubscribePublishUnsubscribeMultiCommon(t, s, "enigma", pubMessage, false, true)
	m := <-pubMessage
	msg := m.(map[string]interface{})
	assert.Equal("12345", msg["not_other"])
	assert.Equal("yay!", msg["pn_other"])

}

func TestSubscribePublishUnsubscribePNOtherComplexPost(t *testing.T) {
	assert := assert.New(t)
	pubMessage := make(chan interface{})
	s1 := map[string]interface{}{
		"id":        1,
		"not_other": "1234567",
	}
	s := map[string]interface{}{
		"id":        1,
		"not_other": "12345",
		"pn_other":  s1,
	}

	go SubscribePublishUnsubscribeMultiCommon(t, s, "enigma", pubMessage, false, true)
	m := <-pubMessage
	msg := m.(map[string]interface{})
	assert.Equal("12345", msg["not_other"])
	if msgOther, ok := msg["pn_other"].(map[string]interface{}); !ok {
		assert.Fail("!map[string]interface{}")
	} else {
		assert.Equal("1234567", msgOther["not_other"])
	}

}

func TestSubscribePublishUnsubscribeInterfaceWithoutPNOtherPost(t *testing.T) {
	assert := assert.New(t)
	pubMessage := make(chan interface{})
	s := map[string]interface{}{
		"id":        3,
		"not_other": "1234",
		"ss_other":  "yay!",
	}

	go SubscribePublishUnsubscribeMultiCommon(t, s, "", pubMessage, false, true)
	m := <-pubMessage
	msg := m.(map[string]interface{})
	assert.Equal("1234", msg["not_other"])
	assert.Equal("yay!", msg["ss_other"])

}

func TestSubscribePublishUnsubscribeInterfaceWithoutPNOtherEncPost(t *testing.T) {
	assert := assert.New(t)
	pubMessage := make(chan interface{})
	s := map[string]interface{}{
		"id":        4,
		"not_other": "123",
		"ss_other":  "yay!",
	}

	go SubscribePublishUnsubscribeMultiCommon(t, s, "enigma", pubMessage, false, true)
	m := <-pubMessage
	msg := m.(map[string]interface{})
	assert.Equal("123", msg["not_other"])
	assert.Equal("yay!", msg["ss_other"])
}

func TestSubscribePublishUnsubscribeInterfaceCustomPost(t *testing.T) {
	assert := assert.New(t)
	pubMessage := make(chan interface{})
	s := customStruct{
		Foo: "hi!",
		Bar: []int{1, 2, 3, 4, 5},
	}

	go SubscribePublishUnsubscribeMultiCommon(t, s, "", pubMessage, false, true)
	m := <-pubMessage
	//s1 := reflect.ValueOf(m)
	//fmt.Println("s:::", s1, s1.Type())
	if msg, ok := m.(map[string]interface{}); !ok {
		//fmt.Println(msg)
		assert.Fail("not map")
	} else {
		//fmt.Println(msg)
		//byt := []byte(message.Message)
		//fmt.Println(message.Message.(string))
		//err := json.Unmarshal(byt, &msg)
		//assert.Nil(err)
		assert.Equal("hi!", msg["Foo"])
		//assert.Equal("1", msg["Bar"].(map[string]interface{})[0])
		//assert.Equal("\"yay!\"", msg["pn_other"])
	}
}

func TestSubscribePublishUnsubscribeInterfaceWithoutCustomEncPost(t *testing.T) {
	assert := assert.New(t)
	pubMessage := make(chan interface{})
	s := customStruct{
		Foo: "hi!",
		Bar: []int{1, 2, 3, 4, 5},
	}

	go SubscribePublishUnsubscribeMultiCommon(t, s, "enigma", pubMessage, false, true)
	m := <-pubMessage
	//s1 := reflect.ValueOf(m)
	//fmt.Println("s:::", s1, s1.Type())
	if msg, ok := m.(map[string]interface{}); !ok {
		//fmt.Println(msg)
		assert.Fail("not map")
	} else {
		//fmt.Println(msg)
		//byt := []byte(message.Message)
		//fmt.Println(message.Message.(string))
		//err := json.Unmarshal(byt, &msg)
		//assert.Nil(err)
		assert.Equal("hi!", msg["Foo"])
		//assert.Equal("1", msg["Bar"].(map[string]interface{})[0])
		//assert.Equal("\"yay!\"", msg["pn_other"])
	}
}

func SubscribePublishUnsubscribeMultiCommon(t *testing.T, s interface{}, cipher string, pubMessage chan interface{}, disablePNOtherProcessing bool, usePost bool) {
	assert := assert.New(t)
	doneSubscribe := make(chan bool)
	doneUnsubscribe := make(chan bool)
	donePublish := make(chan bool)
	errChan := make(chan string)

	//r := GenRandom()

	ch := "testChannel_sub_96112" //fmt.Sprintf("testChannel_sub_%d", r.Intn(99999))

	pn := pubnub.NewPubNub(configCopy())
	pn.Config.CipherKey = cipher
	pn.Config.DisablePNOtherProcessing = disablePNOtherProcessing
	//pn.Config.Log = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

	listener := pubnub.NewListener()

	go func() {
		for {
			select {
			case status := <-listener.Status:
				switch status.Category {
				case pubnub.PNConnectedCategory:
					doneSubscribe <- true
				case pubnub.PNDisconnectedCategory:
					doneUnsubscribe <- true
				}
			case message := <-listener.Message:
				donePublish <- true
				if pubMessage != nil {
					pubMessage <- message.Message
				} else {
					fmt.Println("pubMessage nil")
				}

			case <-listener.Presence:
				errChan <- "Got presence while awaiting for a status event"
			}
		}
	}()

	pn.AddListener(listener)

	pn.Subscribe().Channels([]string{ch}).Execute()

	select {
	case <-doneSubscribe:
	case err := <-errChan:
		assert.Fail(err)
		return
	}

	pn.Publish().Channel(ch).Message(s).UsePost(usePost).Execute()

	select {
	case <-donePublish:
	case err := <-errChan:
		assert.Fail(err)
		return
	}

	pn.Unsubscribe().
		Channels([]string{ch}).
		Execute()

	select {
	case <-doneUnsubscribe:
	case err := <-errChan:
		assert.Fail(err)
	}

	assert.Zero(len(pn.GetSubscribedChannels()))
	assert.Zero(len(pn.GetSubscribedGroups()))
}

/*func TestSubscribePublishUnsubscribePNOther(t *testing.T) {
	assert := assert.New(t)
	doneSubscribe := make(chan bool)
	doneUnsubscribe := make(chan bool)
	donePublish := make(chan bool)
	errChan := make(chan string)

	//r := GenRandom()

	ch := "testChannel_sub_96112" //fmt.Sprintf("testChannel_sub_%d", r.Intn(99999))

	pn := pubnub.NewPubNub(configCopy())
	pn.Config.CipherKey = "enigma"
	pn.Config.Log = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

	s := map[string]interface{}{
		"id":        1,
		"not_other": "12345",
		"pn_other":  "yay!",
	}
	listener := pubnub.NewListener()

	go func() {
		for {
			select {
			case status := <-listener.Status:
				switch status.Category {
				case pubnub.PNConnectedCategory:
					doneSubscribe <- true
				case pubnub.PNDisconnectedCategory:
					doneUnsubscribe <- true
				}
			case message := <-listener.Message:
				msg := message.Message.(map[string]interface{})
				assert.Equal("12345", msg["not_other"])
				assert.Equal("\"yay!\"", msg["pn_other"])
				donePublish <- true
			case <-listener.Presence:
				errChan <- "Got presence while awaiting for a status event"
			}
		}
	}()

	pn.AddListener(listener)

	pn.Subscribe().Channels([]string{ch}).Execute()

	select {
	case <-doneSubscribe:
	case err := <-errChan:
		assert.Fail(err)
		return
	}

	pn.Publish().Channel(ch).Message(s).Execute()

	select {
	case <-donePublish:
	case err := <-errChan:
		assert.Fail(err)
		return
	}

	pn.Unsubscribe().
		Channels([]string{ch}).
		Execute()

	select {
	case <-doneUnsubscribe:
	case err := <-errChan:
		assert.Fail(err)
	}

	assert.Zero(len(pn.GetSubscribedChannels()))
	assert.Zero(len(pn.GetSubscribedGroups()))
}*/

/*func TestSubscribePublishUnsubscribePNOtherDisable(t *testing.T) {
	assert := assert.New(t)
	doneSubscribe := make(chan bool)
	doneUnsubscribe := make(chan bool)
	donePublish := make(chan bool)
	errChan := make(chan string)

	//r := GenRandom()

	ch := "testChannel_sub_96112" //fmt.Sprintf("testChannel_sub_%d", r.Intn(99999))

	pn := pubnub.NewPubNub(configCopy())
	pn.Config.CipherKey = "enigma"
	pn.Config.DisablePNOtherProcessing = true
	pn.Config.Log = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

	s := map[string]interface{}{
		"id":        2,
		"not_other": "1234",
		"pn_other":  "\"yay!\"",
	}
	listener := pubnub.NewListener()

	go func() {
		for {
			select {
			case status := <-listener.Status:
				switch status.Category {
				case pubnub.PNConnectedCategory:
					doneSubscribe <- true
				case pubnub.PNDisconnectedCategory:
					doneUnsubscribe <- true
				}
			case message := <-listener.Message:
				//var msg map[string]interface{}
				fmt.Println("reflect.TypeOf(data).Kind()", reflect.TypeOf(message.Message).Kind(), message.Message)
				if msg, ok := message.Message.(map[string]interface{}); !ok {
					fmt.Println(msg)
					assert.Fail("not map")
				} else {
					fmt.Println(msg)
					//byt := []byte(message.Message)
					//fmt.Println(message.Message.(string))
					//err := json.Unmarshal(byt, &msg)
					//assert.Nil(err)
					assert.Equal("1234", msg["not_other"])
					assert.Equal("\"yay!\"", msg["pn_other"])
				}
				donePublish <- true
			case <-listener.Presence:
				errChan <- "Got presence while awaiting for a status event"
			}
		}
	}()

	pn.AddListener(listener)

	pn.Subscribe().Channels([]string{ch}).Execute()

	select {
	case <-doneSubscribe:
	case err := <-errChan:
		assert.Fail(err)
		return
	}

	pn.Publish().Channel(ch).Message(s).Execute()

	select {
	case <-donePublish:
	case err := <-errChan:
		assert.Fail(err)
		return
	}

	pn.Unsubscribe().
		Channels([]string{ch}).
		Execute()

	select {
	case <-doneUnsubscribe:
	case err := <-errChan:
		assert.Fail(err)
	}

	assert.Zero(len(pn.GetSubscribedChannels()))
	assert.Zero(len(pn.GetSubscribedGroups()))
}*/

/*func TestSubscribePublishUnsubscribeInterfaceWithoutPNOther(t *testing.T) {
	assert := assert.New(t)
	doneSubscribe := make(chan bool)
	doneUnsubscribe := make(chan bool)
	donePublish := make(chan bool)
	errChan := make(chan string)

	//r := GenRandom()

	ch := "testChannel_sub_96112" //fmt.Sprintf("testChannel_sub_%d", r.Intn(99999))

	pn := pubnub.NewPubNub(configCopy())

	s := map[string]interface{}{
		"id":        3,
		"not_other": "1234",
		"ss_other":  "\"yay!\"",
	}
	listener := pubnub.NewListener()

	go func() {
		for {
			select {
			case status := <-listener.Status:
				switch status.Category {
				case pubnub.PNConnectedCategory:
					doneSubscribe <- true
				case pubnub.PNDisconnectedCategory:
					doneUnsubscribe <- true
				}
			case message := <-listener.Message:
				var msg map[string]interface{}
				fmt.Println("reflect.TypeOf(data).Kind()", reflect.TypeOf(message.Message).Kind(), message.Message)
				msg = message.Message.(map[string]interface{})
				fmt.Println(msg)
				assert.Equal("1234", msg["not_other"])
				assert.Equal("\"yay!\"", msg["ss_other"])
				donePublish <- true
			case <-listener.Presence:
				errChan <- "Got presence while awaiting for a status event"
			}
		}
	}()

	pn.AddListener(listener)

	pn.Subscribe().Channels([]string{ch}).Execute()

	select {
	case <-doneSubscribe:
	case err := <-errChan:
		assert.Fail(err)
		return
	}

	pn.Publish().Channel(ch).Message(s).Execute()

	select {
	case <-donePublish:
	case err := <-errChan:
		assert.Fail(err)
		return
	}

	pn.Unsubscribe().
		Channels([]string{ch}).
		Execute()

	select {
	case <-doneUnsubscribe:
	case err := <-errChan:
		assert.Fail(err)
	}

	assert.Zero(len(pn.GetSubscribedChannels()))
	assert.Zero(len(pn.GetSubscribedGroups()))
}*/

/*func TestSubscribePublishUnsubscribeInterfaceWithoutPNOtherEnc(t *testing.T) {
assert := assert.New(t)
doneSubscribe := make(chan bool)
doneUnsubscribe := make(chan bool)
donePublish := make(chan bool)
errChan := make(chan string)

//r := GenRandom()

ch := "testChannel_sub_96112" //fmt.Sprintf("testChannel_sub_%d", r.Intn(99999))

pn := pubnub.NewPubNub(configCopy())
pn.Config.CipherKey = "enigma"
pn.Config.Log = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

/*s := map[string]interface{}{
	"not_other": "1234",
	"ss_other":  "\"yay!\"",
}*/
//s := 1.1
/*s := customStruct{
		Foo: "hi!",
		Bar: []int{1, 2, 3, 4, 5},
	}

	listener := pubnub.NewListener()

	go func() {
		for {
			select {
			case status := <-listener.Status:
				switch status.Category {
				case pubnub.PNConnectedCategory:
					doneSubscribe <- true
				case pubnub.PNDisconnectedCategory:
					doneUnsubscribe <- true
				}
			case message := <-listener.Message:
				fmt.Println("reflect.TypeOf(data).Kind()", reflect.TypeOf(message.Message).Kind(), message.Message)
				s := reflect.ValueOf(message.Message)
				fmt.Println("s:::", s, s.Type())
				if msg, ok := message.Message.(map[string]interface{}); !ok {
					fmt.Println(msg)
					assert.Fail("not map")
				} else {
					fmt.Println(msg)
					//byt := []byte(message.Message)
					//fmt.Println(message.Message.(string))
					//err := json.Unmarshal(byt, &msg)
					//assert.Nil(err)
					assert.Equal("hi!", msg["Foo"])
					//assert.Equal("1", msg["Bar"].(map[string]interface{})[0])
					//assert.Equal("\"yay!\"", msg["pn_other"])
				}
				donePublish <- true
			case <-listener.Presence:
				errChan <- "Got presence while awaiting for a status event"
			}
		}
	}()

	pn.AddListener(listener)

	pn.Subscribe().Channels([]string{ch}).Execute()

	select {
	case <-doneSubscribe:
	case err := <-errChan:
		assert.Fail(err)
		return
	}

	pn.Publish().Channel(ch).Message(s).Execute()

	select {
	case <-donePublish:
	case err := <-errChan:
		assert.Fail(err)
		return
	}

	pn.Unsubscribe().
		Channels([]string{ch}).
		Execute()

	select {
	case <-doneUnsubscribe:
	case err := <-errChan:
		assert.Fail(err)
	}

	assert.Zero(len(pn.GetSubscribedChannels()))
	assert.Zero(len(pn.GetSubscribedGroups()))
}*/

/*func TestSubscribePublishUnsubscribe(t *testing.T) {
	assert := assert.New(t)
	doneSubscribe := make(chan bool)
	doneUnsubscribe := make(chan bool)
	donePublish := make(chan bool)
	errChan := make(chan string)
	ch := randomized("sub-pu-ch")

	pn := pubnub.NewPubNub(configCopy())

	listener := pubnub.NewListener()

	go func() {
		for {
			select {
			case status := <-listener.Status:
				switch status.Category {
				case pubnub.PNConnectedCategory:
					doneSubscribe <- true
				case pubnub.PNDisconnectedCategory:
					doneUnsubscribe <- true
				}
			case message := <-listener.Message:
				assert.Equal(message.Message, "hey")
				donePublish <- true
			case <-listener.Presence:
				errChan <- "Got presence while awaiting for a status event"
			}
		}
	}()

	pn.AddListener(listener)

	pn.Subscribe().Channels([]string{ch}).Execute()

	select {
	case <-doneSubscribe:
	case err := <-errChan:
		assert.Fail(err)
		return
	}

	pn.Publish().Channel(ch).Message("hey").Execute()

	select {
	case <-donePublish:
	case err := <-errChan:
		assert.Fail(err)
		return
	}

	pn.Unsubscribe().
		Channels([]string{ch}).
		Execute()

	select {
	case <-doneUnsubscribe:
	case err := <-errChan:
		assert.Fail(err)
	}

	assert.Zero(len(pn.GetSubscribedChannels()))
	assert.Zero(len(pn.GetSubscribedGroups()))
}*/

// Also tests:
// - test operations like publish/unsubscribe invoked inside another goroutine
// - test unsubscribe all
func TestSubscribePublishPartialUnsubscribe(t *testing.T) {
	assert := assert.New(t)
	doneUnsubscribe := make(chan bool)
	errChan := make(chan string)
	var once sync.Once

	ch1 := randomized("sub-partialu-ch1")
	ch2 := randomized("sub-partialu-ch2")
	heyPub := heyIterator(3)
	heySub := heyIterator(3)

	pn := pubnub.NewPubNub(configCopy())
	pn.Config.Uuid = randomized("sub-partialu-uuid")

	listener := pubnub.NewListener()

	go func() {
		for {
			select {
			case status := <-listener.Status:
				switch status.Category {
				case pubnub.PNConnectedCategory:
					once.Do(func() {
						pn.Publish().Channel(ch1).Message(<-heyPub).Execute()
					})
					continue
				}
				if len(status.AffectedChannels) == 1 && status.Operation == pubnub.PNUnsubscribeOperation {
					assert.Equal(status.AffectedChannels[0], ch2)
					doneUnsubscribe <- true
				}
			case message := <-listener.Message:
				if message.Message == <-heySub {
					pn.Unsubscribe().
						Channels([]string{ch2}).
						Execute()
				} else {
					errChan <- fmt.Sprintf("Unexpected message: %s",
						message.Message)
				}
			case <-listener.Presence:
				errChan <- "Got presence while awaiting for a status event"
			}
		}
	}()

	pn.AddListener(listener)

	pn.Subscribe().Channels([]string{ch1, ch2}).Execute()

	select {
	case <-doneUnsubscribe:
	case err := <-errChan:
		assert.Fail(err)
	}

	pn.RemoveListener(listener)
	pn.UnsubscribeAll()

	assert.Zero(len(pn.GetSubscribedChannels()))
	assert.Zero(len(pn.GetSubscribedGroups()))
}

func JoinLeaveChannel(t *testing.T) {
	assert := assert.New(t)

	// await both connected event on emitter and join presence event received
	var wg sync.WaitGroup
	wg.Add(2)

	donePresenceConnect := make(chan bool)
	doneJoin := make(chan bool)
	doneLeave := make(chan bool)
	errChan := make(chan string)
	ch := randomized("ch")

	configEmitter := configCopy()
	configPresenceListener := configCopy()

	configEmitter.Uuid = randomized("sub-lj-emitter")
	configPresenceListener.Uuid = randomized("sub-lj-listener")

	pn := pubnub.NewPubNub(configEmitter)
	pnPresenceListener := pubnub.NewPubNub(configPresenceListener)

	listenerEmitter := pubnub.NewListener()
	listenerPresenceListener := pubnub.NewListener()

	// emitter
	go func() {
		for {
			select {
			case status := <-listenerEmitter.Status:
				switch status.Category {
				case pubnub.PNConnectedCategory:
					wg.Done()
					return
				}
			case <-listenerEmitter.Message:
				errChan <- "Got message while awaiting for a status event"
				return
			case <-listenerEmitter.Presence:
				errChan <- "Got presence while awaiting for a status event"
				return
			}
		}
	}()

	// listener
	go func() {
		for {
			select {
			case status := <-listenerPresenceListener.Status:
				switch status.Category {
				case pubnub.PNConnectedCategory:
					donePresenceConnect <- true
				}
			case message := <-listenerPresenceListener.Message:
				errChan <- fmt.Sprintf("Unexpected message: %s",
					message.Message)
			case presence := <-listenerPresenceListener.Presence:
				// ignore join event of presence listener
				if presence.Uuid == configPresenceListener.Uuid {
					continue
				}

				assert.Equal(ch, presence.Channel)

				if presence.Event == "leave" {
					assert.Equal(configEmitter.Uuid, presence.Uuid)
					doneLeave <- true
					return
				} else {
					assert.Equal("join", presence.Event)
					assert.Equal(configEmitter.Uuid, presence.Uuid)
					wg.Done()
				}
			}
		}
	}()

	pn.AddListener(listenerEmitter)
	pnPresenceListener.AddListener(listenerPresenceListener)

	pnPresenceListener.Subscribe().
		Channels([]string{ch}).
		WithPresence(true).
		Execute()

	select {
	case <-donePresenceConnect:
	case err := <-errChan:
		assert.Fail(err)
		return
	}

	pn.Subscribe().
		Channels([]string{ch}).
		Execute()

	go func() {
		wg.Wait()
		doneJoin <- true
	}()

	select {
	case <-doneJoin:
	case err := <-errChan:
		assert.Fail(err)
		return
	}

	pn.Unsubscribe().
		Channels([]string{ch}).
		Execute()

	select {
	case <-doneLeave:
	case err := <-errChan:
		assert.Fail(err)
		return
	}
}

/////////////////////////////
// Channel Group Subscription
/////////////////////////////

func TestSubscribeUnsubscribeGroup(t *testing.T) {
	assert := assert.New(t)
	doneSubscribe := make(chan bool)
	doneUnsubscribe := make(chan bool)
	errChan := make(chan string)
	ch := randomized("sub-sug-ch")
	cg := randomized("sub-sug-cg")

	pn := pubnub.NewPubNub(configCopy())

	listener := pubnub.NewListener()

	go func() {
		for {
			select {
			case status := <-listener.Status:
				switch status.Category {
				case pubnub.PNConnectedCategory:
					doneSubscribe <- true
				case pubnub.PNDisconnectedCategory:
					doneUnsubscribe <- true
					return
				}
			case <-listener.Message:
				errChan <- "Got message while awaiting for a status event"
				return
			case <-listener.Presence:
				errChan <- "Got presence while awaiting for a status event"
				return
			}
		}
	}()

	pn.AddListener(listener)

	_, _, err := pn.AddChannelToChannelGroup().
		Channels([]string{ch}).
		ChannelGroup(cg).
		Execute()

	assert.Nil(err)

	// await for adding channels
	time.Sleep(3 * time.Second)

	pn.Subscribe().
		ChannelGroups([]string{cg}).
		Execute()

	select {
	case <-doneSubscribe:
	case err := <-errChan:
		assert.Fail(err)
	}

	pn.Unsubscribe().
		ChannelGroups([]string{cg}).
		Execute()

	select {
	case <-doneUnsubscribe:
	case err := <-errChan:
		assert.Fail(err)
	}

	assert.Zero(len(pn.GetSubscribedChannels()))
	assert.Zero(len(pn.GetSubscribedGroups()))

	_, _, err = pn.RemoveChannelFromChannelGroup().
		Channels([]string{ch}).
		ChannelGroup(cg).
		Execute()
}

func TestSubscribePublishUnsubscribeAllGroup(t *testing.T) {
	assert := assert.New(t)
	pn := pubnub.NewPubNub(configCopy())
	listener := pubnub.NewListener()
	doneSubscribe := make(chan bool)
	donePublish := make(chan bool)
	doneUnsubscribe := make(chan bool)
	errChan := make(chan string)
	ch := randomized("sub-spuag-ch")
	cg1 := randomized("sub-spuag-cg1")
	cg2 := randomized("sub-spuag-cg2")

	pn.AddListener(listener)

	go func() {
		for {
			select {
			case status := <-listener.Status:
				switch status.Category {
				case pubnub.PNConnectedCategory:
					doneSubscribe <- true
				case pubnub.PNDisconnectedCategory:
					doneUnsubscribe <- true
				}
			case message := <-listener.Message:
				donePublish <- true
				assert.Equal("hey", message.Message)
				assert.Equal(ch, message.Channel)
			case <-listener.Presence:
				errChan <- "Got presence while awaiting for a status event"
			}
		}
	}()

	_, _, err := pn.AddChannelToChannelGroup().
		Channels([]string{ch}).
		ChannelGroup(cg1).
		Execute()

	assert.Nil(err)

	// await for adding channel
	time.Sleep(2 * time.Second)

	pn.Subscribe().
		ChannelGroups([]string{cg1, cg2}).
		Execute()

	select {
	case <-doneSubscribe:
	case err := <-errChan:
		assert.Fail(err)
		return
	}

	pn.Publish().Channel(ch).Message("hey").Execute()

	select {
	case <-donePublish:
	case err := <-errChan:
		assert.Fail(err)
		return
	}

	pn.Unsubscribe().
		ChannelGroups([]string{cg2}).
		Execute()

	assert.Equal(len(pn.GetSubscribedGroups()), 1)

	pn.UnsubscribeAll()

	select {
	case <-doneUnsubscribe:
	case err := <-errChan:
		assert.Fail(err)
		return
	}

	assert.Equal(len(pn.GetSubscribedGroups()), 0)

	_, _, err = pn.RemoveChannelFromChannelGroup().
		Channels([]string{ch}).
		ChannelGroup(cg1).
		Execute()

	assert.Nil(err)
}

func SubscribeJoinLeaveGroup(t *testing.T) {
	assert := assert.New(t)

	// await both connected event on emitter and join presence event received
	var wg sync.WaitGroup
	wg.Add(2)

	donePresenceConnect := make(chan bool)
	doneJoinEvent := make(chan bool)
	doneLeaveEvent := make(chan bool)
	errChan := make(chan string)
	ch := randomized("sub-jlg-ch")
	cg := randomized("sub-jlg-cg")

	configEmitter := configCopy()
	configPresenceListener := configCopy()

	configEmitter.Uuid = randomized("emitter")
	configPresenceListener.Uuid = randomized("listener")

	pn := pubnub.NewPubNub(configEmitter)
	pnPresenceListener := pubnub.NewPubNub(configPresenceListener)

	listenerEmitter := pubnub.NewListener()
	listenerPresenceListener := pubnub.NewListener()

	// emitter
	go func() {
		for {
			select {
			case status := <-listenerEmitter.Status:
				switch status.Category {
				case pubnub.PNConnectedCategory:
					wg.Done()
					return
				}
			case <-listenerEmitter.Message:
				errChan <- "Got message while awaiting for a status event"
				return
			case <-listenerEmitter.Presence:
				errChan <- "Got presence while awaiting for a status event"
				return
			}
		}
	}()

	// listener
	go func() {
		for {
			select {
			case status := <-listenerPresenceListener.Status:
				switch status.Category {
				case pubnub.PNConnectedCategory:
					donePresenceConnect <- true
				}
			case <-listenerPresenceListener.Message:
				errChan <- "Got message while awaiting for a status event"
				return
			case presence := <-listenerPresenceListener.Presence:
				// ignore join event of presence listener
				if presence.Uuid == configPresenceListener.Uuid {
					continue
				}

				assert.Equal(presence.Channel, ch)

				if presence.Event == "leave" {
					assert.Equal(configEmitter.Uuid, presence.Uuid)
					doneLeaveEvent <- true
					return
				} else {
					assert.Equal("join", presence.Event)
					assert.Equal(configEmitter.Uuid, presence.Uuid)
					wg.Done()
				}
			}
		}
	}()

	pn.AddListener(listenerEmitter)
	pnPresenceListener.AddListener(listenerPresenceListener)

	pnPresenceListener.AddChannelToChannelGroup().
		Channels([]string{ch}).
		ChannelGroup(cg).
		Execute()

	pnPresenceListener.Subscribe().
		ChannelGroups([]string{cg}).
		WithPresence(true).
		Execute()

	select {
	case <-donePresenceConnect:
	case err := <-errChan:
		assert.Fail(err)
	}

	pn.Subscribe().
		ChannelGroups([]string{cg}).
		Execute()

	go func() {
		wg.Wait()
		doneJoinEvent <- true
	}()

	select {
	case <-doneJoinEvent:
	case err := <-errChan:
		assert.Fail(err)
	}

	pn.Unsubscribe().
		ChannelGroups([]string{cg}).
		Execute()

	select {
	case <-doneLeaveEvent:
	case err := <-errChan:
		assert.Fail(err)
	}
}

/////////////////////////////
// Unsubscribe
/////////////////////////////

func TestUnsubscribeAll(t *testing.T) {
	assert := assert.New(t)
	pn := pubnub.NewPubNub(configCopy())
	channels := []string{
		randomized("sub-ua-ch1"),
		randomized("sub-ua-ch2"),
		randomized("sub-ua-ch3")}

	groups := []string{
		randomized("sub-ua-cg1"),
		randomized("sub-ua-cg2"),
		randomized("sub-ua-cg3")}

	pn.Subscribe().
		Channels(channels).
		ChannelGroups(groups).
		WithPresence(true).
		Execute()

	assert.Equal(len(pn.GetSubscribedChannels()), 3)
	assert.Equal(len(pn.GetSubscribedGroups()), 3)

	pn.UnsubscribeAll()

	assert.Equal(len(pn.GetSubscribedChannels()), 0)
	assert.Equal(len(pn.GetSubscribedGroups()), 0)
}

/////////////////////////////
// Misc
/////////////////////////////

func TestSubscribe403Error(t *testing.T) {
	assert := assert.New(t)
	doneSubscribe := make(chan bool)
	doneAccessDenied := make(chan bool)
	errChan := make(chan string)

	pn := pubnub.NewPubNub(pamConfigCopy())
	listener := pubnub.NewListener()

	go func() {
		for {
			select {
			case status := <-listener.Status:
				switch status.Category {
				case pubnub.PNConnectedCategory:
					doneSubscribe <- true
				case pubnub.PNAccessDeniedCategory:
					doneAccessDenied <- true
				}
			case <-listener.Message:
				errChan <- "Got message while awaiting for a status event"
				return
			case <-listener.Presence:
				errChan <- "Got presence while awaiting for a status event"
				return
			}
		}
	}()

	pn.AddListener(listener)

	pn.Grant().
		Read(false).
		Write(false).
		Manage(false).
		AuthKeys([]string{"pam-key"}).
		Channels([]string{"ch"}).
		Execute()

	pn.Config.SecretKey = ""

	pn.Subscribe().
		Channels([]string{"ch"}).
		Execute()

	select {
	case <-doneSubscribe:
		assert.Fail("Access denied expected")
	case <-doneAccessDenied:
	case err := <-errChan:
		assert.Fail(err)
	}

	assert.Zero(len(pn.GetSubscribedChannels()))
	assert.Zero(len(pn.GetSubscribedGroups()))
}

func TestSubscribeParseUserMeta(t *testing.T) {
	interceptor := stubs.NewInterceptor()
	interceptor.AddStub(&stubs.Stub{
		Method:             "GET",
		Path:               "/v2/subscribe/sub-c-e41d50d4-43ce-11e8-a433-9e6b275e7b64/ch/0",
		Query:              "heartbeat=300",
		ResponseBody:       `{"t":{"t":"14858178301085322","r":7},"m":[{"a":"4","f":512,"i":"02a7b822-220c-49b0-90c4-d9cbecc0fd85","s":1,"p":{"t":"14858178301075219","r":7},"k":"demo-36","c":"chTest","u":"my-data","d":{"City":"Goiania","Name":"Marcelo"}}]}`,
		IgnoreQueryKeys:    []string{"pnsdk", "uuid"},
		ResponseStatusCode: 200,
	})

	assert := assert.New(t)
	doneMeta := make(chan bool)
	errChan := make(chan string)

	pn := pubnub.NewPubNub(configCopy())
	pn.SetSubscribeClient(interceptor.GetClient())
	listener := pubnub.NewListener()

	go func() {
		for {
			select {
			case status := <-listener.Status:
				// ignore status messages
				if status.Error {
					errChan <- fmt.Sprintf("Status Error: %s", status.Category)
				}
			case message := <-listener.Message:
				meta, ok := message.UserMetadata.(string)
				if !ok {
					errChan <- "Invalid message type"
				}

				assert.Equal(meta, "my-data")
				doneMeta <- true
			case <-listener.Presence:
				errChan <- "Got presence while awaiting for a status event"
				return
			}
		}
	}()

	pn.AddListener(listener)

	pn.Subscribe().
		Channels([]string{"ch"}).
		Execute()

	select {
	case <-doneMeta:
	case err := <-errChan:
		assert.Fail(err)
	}
}

func TestSubscribeWithCustomTimetoken(t *testing.T) {
	ch := "ch"
	interceptor := stubs.NewInterceptor()
	interceptor.AddStub(&stubs.Stub{
		Method:             "GET",
		Path:               "/v2/subscribe/sub-c-e41d50d4-43ce-11e8-a433-9e6b275e7b64/ch/0",
		ResponseBody:       `{"t":{"t":"15069659902324693","r":12},"m":[]}`,
		Query:              "heartbeat=300",
		IgnoreQueryKeys:    []string{"pnsdk", "uuid"},
		ResponseStatusCode: 200,
	})
	interceptor.AddStub(&stubs.Stub{
		Method:             "GET",
		Path:               "/v2/subscribe/sub-c-e41d50d4-43ce-11e8-a433-9e6b275e7b64/ch/0",
		ResponseBody:       `{"t":{"t":"14607577960932487","r":1},"m":[{"a":"4","f":0,"i":"Client-g5d4g","p":{"t":"14607577960925503","r":1},"k":"sub-c-e41d50d4-43ce-11e8-a433-9e6b275e7b64","c":"ch","d":{"text":"Enter Message Here"},"b":"ch"}]}`,
		Query:              "heartbeat=300&tt=1337",
		IgnoreQueryKeys:    []string{"pnsdk", "uuid"},
		ResponseStatusCode: 200,
		Hang:               true,
	})

	assert := assert.New(t)
	doneConnected := make(chan bool)
	errChan := make(chan string)

	pn := pubnub.NewPubNub(configCopy())
	pn.SetSubscribeClient(interceptor.GetClient())
	listener := pubnub.NewListener()

	go func() {
		for {
			select {
			case status := <-listener.Status:
				if status.Category == pubnub.PNConnectedCategory {
					doneConnected <- true
				} else {
					errChan <- fmt.Sprintf("Got status while awaiting for a message: %s",
						status.Category)
					return
				}
			case <-listener.Message:
				errChan <- "Got message while awaiting for a message"
			case <-listener.Presence:
				errChan <- "Got presence while awaiting for a message"
				return
			}
		}
	}()

	pn.AddListener(listener)

	pn.Subscribe().
		Channels([]string{ch}).
		Timetoken(int64(1337)).
		Execute()

	select {
	case <-doneConnected:
	case err := <-errChan:
		assert.Fail(err)
	}

	pn.UnsubscribeAll()
}

func TestSubscribeWithFilter(t *testing.T) {
	assert := assert.New(t)
	doneSubscribe := make(chan bool)
	donePublish := make(chan bool)
	errChan := make(chan string)
	ch := randomized("sub-wf-ch")

	pn := pubnub.NewPubNub(configCopy())
	pn.Config.FilterExpression = "language!=spanish"
	listener := pubnub.NewListener()

	go func() {
		for {
			select {
			case status := <-listener.Status:
				switch status.Category {
				case pubnub.PNConnectedCategory:
					doneSubscribe <- true
				}
			case message := <-listener.Message:
				if message.Message == "Hello!" {
					donePublish <- true
				}
			case <-listener.Presence:
				errChan <- "Got presence while awaiting for a status event"
				return
			}
		}
	}()

	pn.AddListener(listener)

	pn.Subscribe().
		Channels([]string{ch}).
		Execute()

	select {
	case <-doneSubscribe:
	case err := <-errChan:
		assert.Fail(err)
	}

	pnPublish := pubnub.NewPubNub(configCopy())

	meta := make(map[string]string)
	meta["language"] = "spanish"

	pnPublish.Publish().
		Channel("ch").
		Meta(meta).
		Message("Hola!").
		Execute()

	anotherMeta := make(map[string]string)
	anotherMeta["language"] = "english"

	pnPublish.Publish().
		Channel(ch).
		Meta(anotherMeta).
		Message("Hello!").
		Execute()

	<-donePublish
}

func TestSubscribePublishUnsubscribeWithEncrypt(t *testing.T) {
	assert := assert.New(t)
	doneConnect := make(chan bool)
	donePublish := make(chan bool)
	errChan := make(chan string)
	ch := randomized("sub-puwe-ch")

	config := configCopy()
	config.CipherKey = "my-key"
	pn := pubnub.NewPubNub(config)
	listener := pubnub.NewListener()

	go func() {
		for {
			select {
			case status := <-listener.Status:
				switch status.Category {
				case pubnub.PNConnectedCategory:
					doneConnect <- true
				}
			case message := <-listener.Message:
				assert.Equal("hey", message.Message)
				donePublish <- true
			case <-listener.Presence:
				errChan <- "Got presence while awaiting for a status event"
				return
			}
		}
	}()

	pn.AddListener(listener)

	pn.Subscribe().
		Channels([]string{ch}).
		Execute()

	select {
	case <-doneConnect:
	case err := <-errChan:
		assert.Fail(err)
	}

	pn.Publish().
		UsePost(true).
		Channel(ch).
		Message("hey").
		Execute()

	select {
	case <-donePublish:
	case err := <-errChan:
		assert.Fail(err)
	}
}

func TestSubscribeSuperCall(t *testing.T) {
	assert := assert.New(t)
	doneSubscribe := make(chan bool)
	errChan := make(chan string)
	config := pamConfigCopy()
	// Not allowed characters:
	// .,:*
	validCharacters := "-_~?#[]@!$&'()+;=`|"
	config.Uuid = validCharacters
	config.AuthKey = validCharacters

	pn := pubnub.NewPubNub(config)
	listener := pubnub.NewListener()

	go func() {
		for {
			select {
			case status := <-listener.Status:
				switch status.Category {
				case pubnub.PNConnectedCategory:
					doneSubscribe <- true
				}
			case <-listener.Message:
				errChan <- "Got message while awaiting for a status event"
				return
			case <-listener.Presence:
				errChan <- "Got presence while awaiting for a status event"
				return
			}
		}
	}()

	pn.AddListener(listener)

	// Not allowed characters:
	// ?#[]@!$&'()+;=`|
	groupCharacters := "-_~"

	pn.Subscribe().
		Channels([]string{validCharacters + "channel"}).
		ChannelGroups([]string{groupCharacters + "cg"}).
		Timetoken(int64(1337)).
		Execute()

	select {
	case <-doneSubscribe:
	case err := <-errChan:
		assert.Fail(err)
	}
}

func TestReconnectionExhaustion(t *testing.T) {
	assert := assert.New(t)
	doneSubscribe := make(chan bool)
	errChan := make(chan string)

	interceptor := stubs.NewInterceptor()
	interceptor.AddStub(&stubs.Stub{
		Method:             "GET",
		Path:               "/v2/subscribe/sub-c-e41d50d4-43ce-11e8-a433-9e6b275e7b64/ch/0",
		ResponseBody:       "",
		Query:              "heartbeat=300",
		IgnoreQueryKeys:    []string{"pnsdk", "uuid"},
		ResponseStatusCode: 400,
	})

	config.MaximumReconnectionRetries = 1
	config.PNReconnectionPolicy = pubnub.PNLinearPolicy
	pn := pubnub.NewPubNub(config)
	pn.SetSubscribeClient(interceptor.GetClient())
	listener := pubnub.NewListener()

	go func() {
		for {
			select {
			case status := <-listener.Status:
				switch status.Category {
				case pubnub.PNReconnectedCategory:
					doneSubscribe <- true
				}
			case <-listener.Message:
				errChan <- "Got message while awaiting for a status event"
				return
			case <-listener.Presence:
				errChan <- "Got presence while awaiting for a status event"
				return
			}
		}
	}()

	pn.AddListener(listener)

	pn.Subscribe().
		Channels([]string{"ch"}).
		Execute()

	select {
	case <-doneSubscribe:
	case err := <-errChan:
		assert.Fail(err)
	}
}
