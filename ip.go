package scipipe

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// IP Is the base interface which all other IPs need to adhere to
type IP interface {
	ID() string
	FinalizePath()
}

// ------------------------------------------------------------------------
// BaseIP type
// ------------------------------------------------------------------------

// BaseIP contains foundational functionality which all IPs need to implement.
// It is meant to be embedded into other IP implementations.
type BaseIP struct {
	path      string
	id        string
	auditInfo *AuditInfo
}

// NewBaseIP creates a new BaseIP
func NewBaseIP(path string) *BaseIP {
	return &BaseIP{
		path: path,
		id:   randSeqLC(20),
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
	buffer    *bytes.Buffer
	doStream  bool
	lock      *sync.Mutex
	SubStream *InPort
}

// NewFileIP creates a new FileIP
func NewFileIP(path string) (*FileIP, error) {
	isValid, err := pathIsValid(path)
	if err != nil {
		return nil, err
	}
	if !isValid {
		return nil, errors.New(fmt.Sprintf(`Could not create new FileIP with filename "%s". File is invalid (perhaps contains invalid characters)`, path))
	}

	ip := &FileIP{
		BaseIP:    NewBaseIP(path),
		lock:      &sync.Mutex{},
		SubStream: NewInPort("in_substream"),
	}
	if ip.Exists() {
		ip.AuditInfo() // This will populate the audit info from file
	}
	//Don't init buffer if not needed?
	//buf := make([]byte, 0, 128)
	//ip.buffer = bytes.NewBuffer(buf)
	return ip, nil
}

func pathIsValid(path string) (bool, error) {
	expr := `^[0-9A-Za-z\/\.\-_]+$`
	ptn, err := regexp.Compile(expr)
	if err != nil {
		return false, errors.New("Could not compile regex")
	}
	return ptn.MatchString(path), nil
}

// ------------------------------------------------------------------------
// Path stuff
// ------------------------------------------------------------------------

// Path returns the (final) path of the physical file
func (ip *FileIP) Path() string {
	return ip.path
}

// TempDir returns the path to a temporary directory where outputs are written
func (ip *FileIP) TempDir() string {
	return filepath.Dir(ip.TempPath())
}

// TempPath returns the temporary path of the physical file
func (ip *FileIP) TempPath() string {
	path := replaceParentDirsWithPlaceholder(ip.path)
	if path[0] == '/' {
		return FSRootPlaceHolder + path
	}
	return path
}

// FSRootPlaceHolder is a string to use instead of an initial '/', to indicate
// a path that belongs to the absolute root
const FSRootPlaceHolder = "__fsroot__"

// FifoPath returns the path to use when a FIFO file is used instead of a
// normal file
func (ip *FileIP) FifoPath() string {
	return ip.path + ".fifo"
}

// ------------------------------------------------------------------------
// Check-thing stuff
// ------------------------------------------------------------------------

// Size returns the size of an existing file, in bytes
func (ip *FileIP) Size() int64 {
	fi, err := os.Stat(ip.path)
	Check(err)
	return fi.Size()
}

// Exists checks if the file exists (at its final file name)
func (ip *FileIP) Exists() bool {
	exists := false
	ip.lock.Lock()
	if _, err := os.Stat(ip.Path()); err == nil {
		exists = true
	}
	ip.lock.Unlock()
	return exists
}

// TempFileExists checks if the temp-file exists
func (ip *FileIP) TempFileExists() bool {
	tempFileExists := false
	ip.lock.Lock()
	if _, err := os.Stat(ip.TempPath()); err == nil {
		tempFileExists = true
	}
	ip.lock.Unlock()
	return tempFileExists
}

// FifoFileExists checks if the FIFO-file (named pipe file) exists
func (ip *FileIP) FifoFileExists() bool {
	fifoFileExists := false
	ip.lock.Lock()
	if _, err := os.Stat(ip.FifoPath()); err == nil {
		fifoFileExists = true
	}
	ip.lock.Unlock()
	return fifoFileExists
}

func (ip *FileIP) String() string {
	return ip.Path()
}

// ------------------------------------------------------------------------
// Open file-stuff
// ------------------------------------------------------------------------

// Open opens the file and returns a file handle (*os.File)
func (ip *FileIP) Open() *os.File {
	f, err := os.Open(ip.Path())
	CheckWithMsg(err, "Could not open file: "+ip.Path())
	return f
}

// OpenTemp opens the temp file and returns a file handle (*os.File)
func (ip *FileIP) OpenTemp() *os.File {
	f, err := os.Open(ip.TempPath())
	CheckWithMsg(err, "Could not open temp file: "+ip.TempPath())
	return f
}

// ------------------------------------------------------------------------
// FIFO-specific stuff
// ------------------------------------------------------------------------

// CreateFifo creates a FIFO file for the FileIP
func (ip *FileIP) CreateFifo() {
	ip.createDirs("")
	ip.lock.Lock()
	cmd := "mkfifo " + ip.FifoPath()
	Debug.Println("Now creating FIFO with command:", cmd)

	if _, err := os.Stat(ip.FifoPath()); err == nil {
		Warning.Printf("[FileIP:%s] FIFO already exists, so not creating a new one: %s", ip.Path(), ip.FifoPath())
	} else {
		_, err := exec.Command("bash", "-c", cmd).Output()
		CheckWithMsg(err, "Could not execute command: "+cmd)
	}

	ip.lock.Unlock()
}

// RemoveFifo removes the FIFO file, if it exists
func (ip *FileIP) RemoveFifo() {
	// FIXME: Shouldn't we check first whether the fifo exists?
	ip.lock.Lock()
	output, err := exec.Command("bash", "-c", "rm "+ip.FifoPath()).Output()
	CheckWithMsg(err, "Could not delete fifo file: "+ip.FifoPath())
	Debug.Println("Removed FIFO output: ", output)
	ip.lock.Unlock()
}

// ------------------------------------------------------------------------
// Read/Write stuff
// ------------------------------------------------------------------------

// Read reads the whole content of the file and returns the content as a byte
// array
func (ip *FileIP) Read() []byte {
	dat, err := ioutil.ReadFile(ip.Path())
	CheckWithMsg(err, "Could not open file for reading: "+ip.Path())
	return dat
}

// Write writes a byte array ([]byte) to the file's temp file path
func (ip *FileIP) Write(dat []byte) {
	ip.createDirs("")
	err := ioutil.WriteFile(ip.TempPath(), dat, 0644)
	CheckWithMsg(err, "Could not write to temp file: "+ip.TempPath())
}

const (
	finalizePathMaxTries      = 3
	finalizePathBackoffFactor = 4
)

// FinalizePath renames the temporary file name to the final file name, thus enabling
// to separate unfinished, and finished files
func (ip *FileIP) FinalizePath() {
	Debug.Println("FileIP: Finalizing path ", ip.TempPath(), "->", ip.Path())
	doneFinalizingPath := false
	tries := 0

	sleepDurationSec := 1
	for !doneFinalizingPath {
		if ip.TempFileExists() {
			ip.lock.Lock()
			tempPaths, err := filepath.Glob(ip.TempDir() + "/*")
			CheckWithMsg(err, "Could not blog directory: "+ip.TempDir())
			for _, tempPath := range tempPaths {
				origDir := filepath.Dir(ip.TempDir())
				origFileName := filepath.Base(tempPath)
				err := os.Rename(tempPath, origDir+"/"+origFileName)
				CheckWithMsg(err, "Could not rename file: "+ip.TempPath())
			}
			err = os.Remove(ip.TempDir())
			CheckWithMsg(err, "Could not remove temp dir: "+ip.TempDir())
			ip.lock.Unlock()
			doneFinalizingPath = true
			Debug.Println("FileIP: Done finalizing path ", ip.TempPath(), "->", ip.Path())
		} else {
			if tries >= finalizePathMaxTries {
				ip.Failf("Failed to find .tmp file after %d tries, so shutting down: %s\nNote: If this problem persists, it could be a problem with your workflow, that the configured output filename in scipipe doesn't match what is written by the tool.", finalizePathMaxTries, ip.TempPath())
			}
			Warning.Printf("[FileIP:%s] Expected .tmp file missing: %s\nSleeping for %d seconds before checking again ...\n", ip.Path(), ip.TempPath(), sleepDurationSec)
			time.Sleep(time.Duration(sleepDurationSec) * time.Second)
			sleepDurationSec *= finalizePathBackoffFactor
			tries++
		}
	}
}

// ------------------------------------------------------------------------
// Params and tags
// ------------------------------------------------------------------------

// Param returns the parameter named key, from the IPs audit info
func (ip *FileIP) Param(key string) string {
	val, ok := ip.AuditInfo().Params[key]
	if !ok {
		ip.Failf("Could not find parameter %s", key)
	}
	return val
}

// ------------------------------------------------------------------------
// Tags stuff
// ------------------------------------------------------------------------

// Tag returns the tag for the tag with key k from the IPs audit info
func (ip *FileIP) Tag(k string) string {
	v, ok := ip.AuditInfo().Tags[k]
	if !ok {
		Warning.Printf("[FileIP:%s] No such tag: (%s)\n", ip.Path(), k)
		return ""
	}
	return v
}

// Tags returns the audit info's tags
func (ip *FileIP) Tags() map[string]string {
	return ip.AuditInfo().Tags
}

// AddTag adds the tag k with value v
func (ip *FileIP) AddTag(k string, v string) {
	ai := ip.AuditInfo()
	if ai.Tags[k] != "" && ai.Tags[k] != v {
		ip.Failf("Can not add value (%s) to existing tag (%s) with different value (%s)", v, k, ai.Tags[k])
	}
	ai.Tags[k] = v
}

// AddTags adds a map of tags to the IPs audit info
func (ip *FileIP) AddTags(tags map[string]string) {
	for k, v := range tags {
		ip.AddTag(k, v)
	}
}

// ------------------------------------------------------------------------
// AuditInfo stuff
// ------------------------------------------------------------------------

// AuditFilePath returns the file path of the audit info file for the FileIP
func (ip *FileIP) AuditFilePath() string {
	return ip.Path() + ".audit.json"
}

// SetAuditInfo sets the AuditInfo struct for the FileIP
func (ip *FileIP) SetAuditInfo(ai *AuditInfo) {
	ip.lock.Lock()
	ip.auditInfo = ai
	ip.lock.Unlock()
}

// WriteAuditLogToFile writes the audit log to its designated file
func (ip *FileIP) WriteAuditLogToFile() {
	auditInfo := ip.AuditInfo()
	auditInfoJSON, jsonErr := json.MarshalIndent(auditInfo, "", "    ")
	CheckWithMsg(jsonErr, "Could not marshall JSON")
	ip.createDirs("")
	writeErr := ioutil.WriteFile(ip.AuditFilePath(), auditInfoJSON, 0644)
	CheckWithMsg(writeErr, "Could not write audit file: "+ip.Path())
}

// AuditInfo returns the AuditInfo struct for the FileIP
func (ip *FileIP) AuditInfo() *AuditInfo {
	defer ip.lock.Unlock()
	ip.lock.Lock()
	if ip.auditInfo == nil {
		ip.auditInfo = UnmarshalAuditInfoJSONFile(ip.AuditFilePath())
	}
	return ip.auditInfo
}

// UnmarshalAuditInfoJSONFile returns an AuditInfo object from an AuditInfo
// .json file
func UnmarshalAuditInfoJSONFile(fileName string) (auditInfo *AuditInfo) {
	auditInfo = NewAuditInfo()
	auditFileData, readFileErr := ioutil.ReadFile(fileName)
	if readFileErr != nil {
		if os.IsNotExist(readFileErr) {
			Info.Printf("Audit file not found, so not unmarshalling: %s\n", fileName)
		} else {
			Failf("Could not read audit file, which does exist: %s", fileName)
		}
	} else {
		unmarshalErr := json.Unmarshal(auditFileData, auditInfo)
		CheckWithMsg(unmarshalErr, "Could not unmarshal audit log file content: "+fileName)
	}
	return auditInfo
}

// ------------------------------------------------------------------------
// Extra convenience functions
// ------------------------------------------------------------------------

// UnMarshalJSON is a helper function to unmarshal the content of the IPs file
// to the interface v
func (ip *FileIP) UnMarshalJSON(v interface{}) {
	d := ip.Read()
	err := json.Unmarshal(d, v)
	CheckWithMsg(err, "Could not unmarshal content of file: "+ip.Path())
}

// ------------------------------------------------------------------------
// Helper functions
// ------------------------------------------------------------------------

func (ip *FileIP) Failf(msg string, parts ...interface{}) {
	ip.Fail(fmt.Sprintf(msg+"\n", parts...))
}

func (ip *FileIP) Fail(msg interface{}) {
	Failf("[FileIP:%s]: %s", ip.Path(), msg)
}

// CreateDirs creates all directories needed to enable writing the IP to its
// path (or temporary-path). If baseDir is provided, it will be prepended
// before all the IPs own temp path. This is to allow components to create their
// own temporary directory, to create the tasks in.
func (ip *FileIP) createDirs(baseDir string) {
	ipDir := ip.TempDir()
	if baseDir != "" {
		ipDir = baseDir + "/" + ip.TempDir()
	}
	if ip.doStream {
		ipDir = filepath.Dir(ip.FifoPath())
	}
	err := os.MkdirAll(ipDir, 0777)
	if err != nil {
		ip.Failf("Could not create directory: (%s): %s\n", ipDir, err)
	}
}

func sanitizePathFragment(s string) (sanitized string) {
	s = strings.ToLower(s)
	disallowedChars := regexp.MustCompile("[^a-z0-9_\\-\\.]+")
	sanitized = disallowedChars.ReplaceAllString(s, "_")
	return
}
