//
// Copyright © 2011 Guy M. Allard
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package stomp

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func (c *Connection) initializeHeartBeats(ch Headers) (e os.Error) {
	e = nil
	// Client wants Heartbeats ?
	vc, ok := ch.Contains("heart-beat")
	if !ok || vc == "0,0" {
		return e
	}
	// Server wants Heartbeats ?
	vs, ok := c.ConnectResponse.Headers.Contains("heart-beat")
	if !ok || vs == "0,0" {
		return e
	}

	// Work area, may or may not become connection heartbeat data
	w := &heartbeat_data{cx: 0, cy: 0, sx: 0, sy: 0,
		hbs: true, hbr: true, // possible reset later
		sti: 0, rti: 0,
		ls: 0, lr: 0}

	// Client specified values
	cp := strings.Split(vc, ",")
	if len(cp) != 2 { // S/B caught by the server first
		return Error("invalid client heart-beat header: " + vc)
	}
	w.cx, e = strconv.Atoi64(cp[0])
	if e != nil {
		return Error("non-numeric cx heartbeat value: " + cp[0])
	}
	w.cy, e = strconv.Atoi64(cp[1])
	if e != nil {
		return Error("non-numeric cy heartbeat value: " + cp[1])
	}

	// Server specified values
	sp := strings.Split(vs, ",")
	if len(sp) != 2 {
		return Error("invalid server heart-beat header: " + vc)
	}
	w.sx, e = strconv.Atoi64(sp[0])
	if e != nil {
		return Error("non-numeric sx heartbeat value: " + sp[0])
	}
	w.sy, e = strconv.Atoi64(sp[1])
	if e != nil {
		return Error("non-numeric sy heartbeat value: " + sp[1])
	}

	// Check for sending needed
	if w.cx == 0 || w.sy == 0 {
		w.hbs = false // 
	}

	// Check for receiving needed
	if w.sx == 0 || w.cy == 0 {
		w.hbr = false // 
	}

	if !w.hbs && !w.hbr {
		return nil // none required
	}

	c.hbd = w                // OK, we are doing some kind of heartbeating
	ct := time.Nanoseconds() // Prime current time

	if w.hbs { // Finish sender parameters if required
		sm := max(w.cx, w.sy)
		smd := sm / 5                // 5% in ms
		w.sti = 1000000 * (sm - smd) // fudge, ns
		w.ssd = make(chan bool)      // add shutdown channel
		w.ls = ct                    // Best guess at start
		go c.sendTicker()
	}

	if w.hbr { // Finish receiver parameters if required
		rm := max(w.sx, w.cy)
		rmd := rm / 10               // 10% in ms
		w.rti = 1000000 * (rm + rmd) // fudge, ns
		w.rsd = make(chan bool)      // add shutdown channel
		w.lr = ct                    // Best guess at start
		go c.receiveTicker()
	}
	return nil
}

func (c *Connection) sendTicker() {
	ticker := time.NewTicker(c.hbd.sti)
	q := false
	for {
		select {
		case ct := <-ticker.C:
			ld := ct - c.hbd.ls
			if ld > (c.hbd.sti - (c.hbd.sti / 5)) { // swag
				// Send a heartbeat
				f := Frame{"\n", Headers{}, make([]uint8, 0)} // Heartbeat frame
				r := make(chan os.Error)
				c.output <- wiredata{f, r}
				e := <-r
				if e != nil {
					fmt.Printf("Heartbeat Send Failure: %v\n", e)
				}
			}
		case q = <-c.hbd.ssd:
			ticker.Stop()
			break
		}
		if q {
			break
		}
	}
	return
}

func (c *Connection) receiveTicker() {
	ticker := time.NewTicker(c.hbd.rti)
	q := false
	for {
		select {
		case ct := <-ticker.C:
			ld := ct - c.hbd.lr
			if ld > (c.hbd.rti - (c.hbd.rti / 5)) { // swag
				c.Hbrf = true // Flag possible dirty connection
			}
		case q = <-c.hbd.rsd:
			ticker.Stop()
			break
		}
		if q {
			break
		}
	}
	return
}