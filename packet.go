package flowbase

import (
	"fmt"
)

// IP Is the base interface which all other IPs need to adhere to
type IP interface {
	ID() string
}

// ------------------------------------------------------------------------
// Packet type
// ------------------------------------------------------------------------

// Packet contains foundational functionality which all IPs need to implement.
// It is meant to be embedded into other IP implementations.
type Packet struct {
	data      any
	id        string
	auditInfo *AuditInfo
	tags      map[string]string
}

// NewPacket creates a new Packet
func NewPacket(data any) *Packet {
	return &Packet{
		data: data,
		id:   randSeqLC(20),
		tags: make(map[string]string),
	}
}

// ID returns a globally unique ID for the IP
func (ip *Packet) ID() string {
	return ip.id
}

// ------------------------------------------------------------------------
// Tags stuff
// ------------------------------------------------------------------------

// Tag returns the tag for the tag with key k from the IPs audit info
func (ip *Packet) Tag(k string) string {
	v, ok := ip.tags[k]
	if !ok {
		Warning.Printf("[Packet:%s] No such tag: (%s)\n", ip.ID(), k)
		return ""
	}
	return v
}

// Tags returns the audit info's tags
func (ip *Packet) Tags() map[string]string {
	return ip.tags
}

// AddTag adds the tag k with value v
func (ip *Packet) AddTag(k string, v string) {
	if ip.tags[k] != "" && ip.tags[k] != v {
		ip.Failf("Can not add value (%s) to existing tag (%s) with different value (%s)", v, k, ip.tags[k])
	}
	ip.tags[k] = v
}

// AddTags adds a map of tags to the IPs audit info
func (ip *Packet) AddTags(tags map[string]string) {
	for k, v := range tags {
		ip.AddTag(k, v)
	}
}

// ------------------------------------------------------------------------
// Helper functions
// ------------------------------------------------------------------------

func (ip *Packet) Failf(msg string, parts ...interface{}) {
	ip.Fail(fmt.Sprintf(msg+"\n", parts...))
}

func (ip *Packet) Fail(msg interface{}) {
	Failf("[Packet:%s]: %s", ip.ID(), msg)
}
