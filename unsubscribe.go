//
// Copyright © 2011-2013 Guy M. Allard
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

package stompngo

/*
	Unsubscribe from a STOMP subscription. 

	Headers MUST contain a "destination" header key, and for Stomp 1.1+, 
	a "id" header key per the specifications.  The subscription MUST currently
	exist for this session.

	Example:
		// Possible additional Header keys: id.
		h := stompngo.Headers{"destination", "/queue/myqueue"}
		e := c.Unsubscribe(h)
		if e != nil {
			// Do something sane ...
		}

*/
func (c *Connection) Unsubscribe(h Headers) error {
	c.log(UNSUBSCRIBE, "start")
	if !c.connected {
		return ECONBAD
	}
	_, e := checkHeaders(h, c)
	if e != nil {
		return e
	}
	if _, ok := h.Contains("destination"); !ok {
		return EREQDSTUNS
	}
	//	ch := h.Clone()

	c.subsLock.Lock()
	defer c.subsLock.Unlock()
	//
	sid, ok := h.Contains("id")

	switch c.protocol {
	case SPL_10:
		if ok { // User specified 'id'
			if _, p := c.subs[sid]; !p { // subscription does not exist
				return EBADSID
			}
		}
	default:
		if !ok {
			return EUNOSID
		}
		if _, p := c.subs[sid]; !p { // subscription does not exist
			return EBADSID
		}
	}

	e = c.transmitCommon(UNSUBSCRIBE, h) // transmitCommon Clones() the headers
	if e != nil {
		return e
	}
	if ok {
		close(c.subs[sid])
		delete(c.subs, sid)
	}
	c.log(UNSUBSCRIBE, "end")
	return nil
}
