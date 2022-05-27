package flowbase

import (
	"bytes"
	"fmt"
	"sync"
)

// IP Is the base interface which all other IPs need to adhere to
type IP interface {
	ID() string
}

// ------------------------------------------------------------------------
// BaseIP type
// ------------------------------------------------------------------------

// BaseIP contains foundational functionality which all IPs need to implement.
// It is meant to be embedded into other IP implementations.
type BaseIP struct {
	data      any
	id        string
	auditInfo *AuditInfo
	tags      map[string]string
}

// NewBaseIP creates a new BaseIP
func NewBaseIP(data any) *BaseIP {
	return &BaseIP{
		data: data,
		id:   randSeqLC(20),
		tags: make(map[string]string),
	}
}

// ID returns a globally unique ID for the IP
func (ip *BaseIP) ID() string {
	return ip.id
}

// ------------------------------------------------------------------------
// FileIP type
// ------------------------------------------------------------------------

// FileIP (Short for "Information Packet" in Flow-Based Programming terminology)
// contains information and helper methods for a physical file on a normal disk.
type FileIP struct {
	*BaseIP
	buffer *bytes.Buffer
	lock   *sync.Mutex
}

// NewFileIP creates a new FileIP
func NewFileIP(data any) (*FileIP, error) {
	ip := &FileIP{
		BaseIP: NewBaseIP(data),
		lock:   &sync.Mutex{},
	}
	return ip, nil
}

// ------------------------------------------------------------------------
// Tags stuff
// ------------------------------------------------------------------------

// Tag returns the tag for the tag with key k from the IPs audit info
func (ip *FileIP) Tag(k string) string {
	v, ok := ip.tags[k]
	if !ok {
		Warning.Printf("[FileIP:%s] No such tag: (%s)\n", ip.ID(), k)
		return ""
	}
	return v
}

// Tags returns the audit info's tags
func (ip *FileIP) Tags() map[string]string {
	return ip.tags
}

// AddTag adds the tag k with value v
func (ip *FileIP) AddTag(k string, v string) {
	if ip.tags[k] != "" && ip.tags[k] != v {
		ip.Failf("Can not add value (%s) to existing tag (%s) with different value (%s)", v, k, ip.tags[k])
	}
	ip.tags[k] = v
}

// AddTags adds a map of tags to the IPs audit info
func (ip *FileIP) AddTags(tags map[string]string) {
	for k, v := range tags {
		ip.AddTag(k, v)
	}
}

// ------------------------------------------------------------------------
// Helper functions
// ------------------------------------------------------------------------

func (ip *FileIP) Failf(msg string, parts ...interface{}) {
	ip.Fail(fmt.Sprintf(msg+"\n", parts...))
}

func (ip *FileIP) Fail(msg interface{}) {
	Failf("[FileIP:%s]: %s", ip.ID(), msg)
}
