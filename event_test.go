/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	testEvent = Event{
		ID:                    "test",
		DatacenterVPCID:       "vpc-0000000",
		DatacenterRegion:      "eu-west-1",
		DatacenterAccessKey:   "key",
		DatacenterAccessToken: "token",
		NetworkAWSID:          "",
		NetworkSubnet:         "10.0.0.0/16",
	}
)

func waitMsg(ch chan *nats.Msg) (*nats.Msg, error) {
	select {
	case msg := <-ch:
		return msg, nil
	case <-time.After(time.Millisecond * 100):
	}
	return nil, errors.New("timeout")
}

func testSetup() (chan *nats.Msg, chan *nats.Msg) {
	doneChan := make(chan *nats.Msg, 10)
	errChan := make(chan *nats.Msg, 10)

	nc, natsErr = nats.Connect(nats.DefaultURL)
	if natsErr != nil {
		panic("Could not connect to nats")
	}

	nc.ChanSubscribe("network.create.aws.done", doneChan)
	nc.ChanSubscribe("network.create.aws.error", errChan)

	return doneChan, errChan
}

func TestEvent(t *testing.T) {
	completed, errored := testSetup()

	Convey("Given I an event", t, func() {
		Convey("With valid fields", func() {
			valid, _ := json.Marshal(testEvent)
			Convey("When processing the event", func() {
				var e Event
				err := e.Process(valid)

				Convey("It should not error", func() {
					So(err, ShouldBeNil)
					msg, timeout := waitMsg(errored)
					So(msg, ShouldBeNil)
					So(timeout, ShouldNotBeNil)
				})

				Convey("It should load the correct values", func() {
					So(e.ID, ShouldEqual, "test")
					So(e.DatacenterVPCID, ShouldEqual, "vpc-0000000")
					So(e.DatacenterRegion, ShouldEqual, "eu-west-1")
					So(e.DatacenterAccessKey, ShouldEqual, "key")
					So(e.DatacenterAccessToken, ShouldEqual, "token")
					So(e.NetworkAWSID, ShouldEqual, "")
					So(e.NetworkSubnet, ShouldEqual, "10.0.0.0/16")
				})
			})

			Convey("When validating the event", func() {
				var e Event
				e.Process(valid)
				err := e.Validate()

				Convey("It should not error", func() {
					So(err, ShouldBeNil)
					msg, timeout := waitMsg(errored)
					So(msg, ShouldBeNil)
					So(timeout, ShouldNotBeNil)
				})
			})

			Convey("When completing the event", func() {
				var e Event
				e.Process(valid)
				e.Complete()
				Convey("It should produce a network.create.aws.done event", func() {
					msg, timeout := waitMsg(completed)
					So(msg, ShouldNotBeNil)
					So(string(msg.Data), ShouldEqual, string(valid))
					So(timeout, ShouldBeNil)
					msg, timeout = waitMsg(errored)
					So(msg, ShouldBeNil)
					So(timeout, ShouldNotBeNil)
				})
			})

			Convey("When erroring the event", func() {
				log.SetOutput(ioutil.Discard)
				var e Event
				e.Process(valid)
				e.Error(errors.New("error"))
				Convey("It should produce a network.create.aws.error event", func() {
					msg, timeout := waitMsg(errored)
					So(msg, ShouldNotBeNil)
					So(string(msg.Data), ShouldContainSubstring, `"error":"error"`)
					So(timeout, ShouldBeNil)
					msg, timeout = waitMsg(completed)
					So(msg, ShouldBeNil)
					So(timeout, ShouldNotBeNil)
				})
				log.SetOutput(os.Stdout)
			})
		})

		Convey("With no datacenter vpc id", func() {
			testEventInvalid := testEvent
			testEventInvalid.DatacenterVPCID = ""
			invalid, _ := json.Marshal(testEventInvalid)

			Convey("When validating the event", func() {
				var e Event
				e.Process(invalid)
				err := e.Validate()
				Convey("It should error", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldEqual, "Datacenter VPC ID invalid")
				})
			})
		})

		Convey("With no datacenter region", func() {
			testEventInvalid := testEvent
			testEventInvalid.DatacenterRegion = ""
			invalid, _ := json.Marshal(testEventInvalid)

			Convey("When validating the event", func() {
				var e Event
				e.Process(invalid)
				err := e.Validate()
				Convey("It should error", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldEqual, "Datacenter Region invalid")
				})
			})
		})

		Convey("With no datacenter access key", func() {
			testEventInvalid := testEvent
			testEventInvalid.DatacenterAccessKey = ""
			invalid, _ := json.Marshal(testEventInvalid)

			Convey("When validating the event", func() {
				var e Event
				e.Process(invalid)
				err := e.Validate()
				Convey("It should error", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldEqual, "Datacenter credentials invalid")
				})
			})
		})

		Convey("With no datacenter access token", func() {
			testEventInvalid := testEvent
			testEventInvalid.DatacenterAccessToken = ""
			invalid, _ := json.Marshal(testEventInvalid)

			Convey("When validating the event", func() {
				var e Event
				e.Process(invalid)
				err := e.Validate()
				Convey("It should error", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldEqual, "Datacenter credentials invalid")
				})
			})
		})

		Convey("With no network subnet", func() {
			testEventInvalid := testEvent
			testEventInvalid.NetworkSubnet = ""
			invalid, _ := json.Marshal(testEventInvalid)

			Convey("When validating the event", func() {
				var e Event
				e.Process(invalid)
				err := e.Validate()
				Convey("It should error", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldEqual, "Network subnet invalid")
				})
			})
		})

	})
}