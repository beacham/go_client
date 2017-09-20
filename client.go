package main

import (
    "net/http"
    "net/http/httputil" // for DumpRequestOut
    "encoding/xml"
    "bytes"
    "io/ioutil"
    "log"
    "time"
    "fmt"
    "os"
    //"strings"
    "crypto/tls"
    //"crypto/x509"
    "regexp" // needed to "fix" the self closing xml tag used by the API, which go does not support
)

var (
    g_SessionId string = ""
)

var (
    httpClient *http.Client
)

const (
    MaxIdleConnections int = 20 // default 20
    RequestTimeout     int = 5  // default 5s
)

const (
        // A generic XML header suitable for use with the output of Marshal.
        // This is not automatically added to any output of this package,
        // it is provided as a convenience.
        Header = `<?xml version="1.0" encoding="UTF-8"?>` + "\n"
)

const (
    //var endPoint string = "http://localhost:8080/doSomething"
    //var endPoint string = "https://10.77.6.12" // real Me7k in San Diego
    //var endPoint string = "https://www.arris.com" // real Me7k (name is configured in /etc/host to map to 10.77.6.12)
    g_EndPoint string = "https://10.10.55.163/neoreq/" // real Me7k in horsham
    //var endPoint string = "http://192.168.0.28:8080" // local server for testing
)

//
// init HTTPClient
//

func init() {
    httpClient = createHTTPClient()
}

//
// createHTTPClient for connection re-use
//

func createHTTPClient() *http.Client {
    client := &http.Client{
        Transport: &http.Transport{
            MaxIdleConnsPerHost: MaxIdleConnections,
            TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
        },
        Timeout: time.Duration(RequestTimeout) * time.Second,
    }

    return client
}

//
// Global Structures for Transcoder
//

/*
 * Format of Login request:
 *
 * <request id="G1000" origin="gui" destination="device" command="add" category="login"
 * time="2015-03-09T17:38:09.783-06:00" protocol- version=“2.3” platform-name="neo">
 * <user name="Admin" password="" type="push"/>
 * </request>
 */

type User struct {
    XMLName   xml.Name `xml:"user"`          // XML tag
    Name     string    `xml:"name,attr"`     // required
    Password string    `xml:"password,attr"` // required
    Type     string    `xml:"type,attr"`     // required
}

type LoginRequest struct {
    XMLName xml.Name   `xml:"request"`
    Id          string `xml:"id,attr"`
    Origin      string `xml:"origin,attr"`
    Destination string `xml:"destination,attr"`
    Command     string `xml:"command,attr"`
    Category    string `xml:"category,attr"`
    Time        string `xml:"time,attr"`     // how do i get a timestamp?
    Version     string `xml:"protocol-version,attr"`
    Platform    string `xml:"platform-name,attr"`
    User        User   // struct
}

/*
 * Format of Login response:
 *
 * <response id="G1000" origin="device" destination="gui" command="add" category="login"
 * time="2015-03-09T17:38:09.783-06:00" protocol- version=“2.3” platform-name="neo" sw-
 * version="me7k.2.3.0" sw-build="1">
 * <session sid="949098745790" type="push" activity-timeout="300000" auth-method="local"
 * farmer-id="Neo-180" client-ip="10.45.0.154" warning="Client time is ahead of controller
 * time"/>
 * </response>
 */

/* Note:
2017/08/30 16:22:27 main - Response Body:
 <?xml version="1.0" encoding="UTF-8"?><response id="beacham" origin="device" destination="transcoder-collector" command="add" category="login" time="2017-08-30T23:24:54.900Z" protocol-version="2.1" platform-name="neo" sw-version="me7k.2.1.2" sw-build="0" status="error"><reason error-code="Unknown_Error"><![CDATA[Number of sessions exceeded the maximum of 8.]]></reason></response>
*/

type LoginResponse struct {
    XMLName xml.Name   `xml:"response"`
    Id          string `xml:"id,attr"`
    Origin      string `xml:"origin,attr"`
    Destination string `xml:"destination,attr"`
    Command     string `xml:"command,attr"`
    Category    string `xml:"category,attr"`
    Time        string `xml:"time,attr"`
    Version     string `xml:"protocol-version,attr"`
    Platform    string `xml:"platform-name,attr"`
    SwVersion   string `xml:"sw-version,attr"`
    SwBuild     string `xml:"sw-build,attr"`
    Session     Session `xml:"session"`// struct
}

type Session struct {
    XMLName         xml.Name `xml:"session"`
    SessionId       string   `xml:"sid,attr"`
    Type            string   `xml:"type,attr"`
    ActivityTimeout string   `xml:"activity-timeout,attr"`
    AuthMethod      string   `xml:"auth-method,attr"`
    FarmerId        string   `xml:"farmer-id,attr"`
    ClientIp        string   `xml:"client-ip,attr"`
    Warning         string   `xml:"warning,attr"`
}

//
// While most event subscriptions are for device-wide events, the subscription for bit rate events must
// be directed to either a line, a mux, or a program level target.
//

/*
 * Format of add subscription (bit rate) request:
 *
 * <request id="G1201" origin="gui" destination="device" command="add" category="subscription"
 * time="2016-02-09T11:30:20.508-06:00" protocol-2.3version=“2.3” platform-name="neo"
 * sid="9223370581815793597">
 * <path>
 * <farmer id="ME-7000-2"/>
 * <board id="4"/>
 * <gige-line id="4/3"/>
 * <gige-output-mux id="0014"/>
 * <output-program id="1"/>
 * </path>
 * <event-list>
 * <event type="bit-rate-event" get-streams="true" get-std-dev="true" get-inst-br="true"
 * get-avg-br="true" get-video-info="true" get-audio-info="true"/>
 * </event-list>
 * </request>
 */

type BitRateRequest struct {
    XMLName xml.Name   `xml:"request"`
    Id          string `xml:"id,attr"`
    Origin      string `xml:"origin,attr"`
    Destination string `xml:"destination,attr"`
    Command     string `xml:"command,attr"`
    Category    string `xml:"category,attr"`
    Time        string `xml:"time,attr"`
    Version     string `xml:"protocol-version,attr"`
    Platform    string `xml:"platform-name,attr"`
    SessionId   string `xml:"sid,attr"`
    Path  Path   // struct
    Event Event  // struct
}

// This struct is awful. Unfortunately, at the time of development golang does not support self closing tags.
// Consequently, this convoluted struct and its siblings are necessary because the me7k uses self closing
// tags rather than explicit closing tags.

type PathLine struct {}
type PathIoMx struct {}
type PathProg struct {}

type Path struct {
        XMLName xml.Name            `xml:"path"` // XML tag
        Farmer  Farmer              `xml:"farmer"`
        Board   Board               `xml:"board"`
        GigeLine GigeLine           `xml:"gige-line"`
        GigeOutputMux GigeOutputMux `xml:"gige-output-mux"`
        //OutputProgram OutputProgram `xml:"output-program"`
}

type Farmer struct {
    XMLName xml.Name `xml:"farmer"` // XML tag
    FarmerId string  `xml:"id,attr"`
}

type Board struct {
    XMLName xml.Name `xml:"board"` // XML tag
    BoardId string   `xml:"id,attr"`
}

type GigeLine struct {
    XMLName xml.Name  `xml:"gige-line"` // XML tag
    GigeLineId string `xml:"id,attr"`
}

type GigeOutputMux struct {
    XMLName xml.Name       `xml:"gige-output-mux"` // XML tag
    GigeOutputMuxId string `xml:"id,attr"`
}

/*
type OutputProgram struct {
    XMLName xml.Name       `xml:"output-program"` // XML tag
    OutputProgramId string `xml:"id,attr"`
}
*/

// This struct is awful. Unfortunately, at the time of development golang does not support self closing tags.
// Consequently, this convoluted struct and its siblings are necessary because the me7k uses self closing
// tags rather than explicit closing tags.

type Event struct {
        XMLName      xml.Name     `xml:"event-list"` // XML tag
        EventBitRate EventBitRate `xml:"event"` // struct
}

type EventBitRate struct {
    XMLName    xml.Name `xml:"event"` // XML element tag
    Type       string   `xml:"type,attr"`
    GetStreams string   `xml:"get-streams,attr"`
    GetStdDev  string   `xml:"get-std-dev,attr"`
    GetInstBr  string   `xml:"get-inst-br,attr"`
    GetAvgBr   string   `xml:"get-avg-br,attr"`
    //GetVideoInfo string   `xml:"get-video-info,attr"`
    //GetAudioInfo string   `xml:"get-audio-info,attr"`
}

/*
* Format of remove subscription (bit rate) request: (Note: Identical to "add" so reuse the struct)
 *
 * <request id="G1204" origin="gui" destination="device" command="remove"
 * category="subscription" time="2016-02-09T11:32:20.177-06:00" protocol-2.3version=“2.3”
 * platform-name="neo" sid="9223370581815793597">
 * <path>
 * <farmer id="ME-7000-2"/>
 * <board id="4"/>
 * <gige-line id="4/3"/>
 * <gige-output-mux id="0014"/>
 * <output-program id="1"/>
 * </path>
 * <event-list>
 * <event type="bit-rate-event"/>
 * </event-list>
 * </request>
 *
 */

/*
 * Format of remove subscription (bit rate) request: (Note: Identical to "add" so reuse the struct)
 *
 * <response id="G1204" origin="device" destination="gui" command="remove"
 * category="subscription" time="2016-02-09T17:32:20.256Z" protocol-version=“2.3” platform-
 * name="neo" sw-version="me7k.2.2.0" sw-build="1" sid="9223370581815793597">
 * <reason error-code="OK"><![CDATA[unsuscribe completed.]]></reason>
 * </response>
 */

/*
 * Format of Get Event Request: (this is used only for sessions of type PULL)
 *
 * <request id='G1111' origin='gui' destination='device' command='get' category='event'
 * time='Fri 2015-06-26T16:37:59.491Z ' protocol- version='1.0' platform-name='neo'
 * sid='1435336628026'/>
 */

type EventRequest struct {
    XMLName xml.Name   `xml:"request"`
    Id          string `xml:"id,attr"`
    Origin      string `xml:"origin,attr"`
    Destination string `xml:"destination,attr"`
    Command     string `xml:"command,attr"`
    Category    string `xml:"category,attr"`
    Time        string `xml:"time,attr"`
    Version     string `xml:"protocol-version,attr"`
    Platform    string `xml:"platform-name,attr"`
    SessionId   string `xml:"sid,attr"`
}

/*
 * Format of Get Event Response: (this is used only for sessions of type PULL)
 *
 * <response id="G1111" origin="device" destination="gui" command="get" category="event"
 * time="2015-06-26T16:37:59.692Z" protocol-version=“2.3” platform-name="neo" sw-
 * version="me7k.1.0.1" sw-build="1" pending-events="0">
 * <event-list><event type="alarm-deleted-event" id="1435265291353"/>
 * <event type="alarm-cleared-event" id="1435316297245" cleared- time="2015-06-
 * 26T16:38:02.513Z"/>
 * <event type="alarm-deleted-event" id="1435265291399"/>
 * <event type="alarm-cleared-event" id="1435316297244" cleared- time="2015-06-
 * 26T16:38:02.513Z"/>
 * <event type="alarm-deleted-event" id="1435265291404"/>
 * </event-list>
 * </response>
 */

/*
 * Format of Get Event Response: (when subscribed to bit rate event at mux level)
 *
 * <response id="beacham" origin="device" destination="transcoder-collector" command="get"
 * category="event" time="2017-09-19T22:15:45.286Z" protocol-version="2.1" platform-name="neo"
 * sw-version="me7k.2.1.2" sw-build="0" pending-events="0"><event-list><event type="bit-rate-event"
 * id="1505838270680" time="2017-09-19T22:15:44.879Z">
 * <path>
 *   <farmer id="ME-7000-1"/>
 *   <board id="4"/>
 *   <gige-line id="4/3"/>
 *   <gige-output-mux id="0000"/>
 * </path>
 * <gige-output-mux id="0000" avg-bit-rate="0" inst-bit-rate="0" overhead="0">
 *   <output-program id="1" avg-bit-rate="0" inst-bit-rate="0">
 *     <stream id="32" avg-bit-rate="0" inst-bit-rate="0" std-dev="0"/>
 *     <stream id="33" avg-bit-rate="0" inst-bit-rate="0" std-dev="0"/>
 *     <stream id="34" avg-bit-rate="0" inst-bit-rate="0" std-dev="0"/>
 *   </output-program>
 *    <passed-pids id="65536" avg-bit-rate="0" inst-bit-rate="0"/>
 *   </gige-output-mux>
 * </event></event-list></response>
 */

type EventResponse struct {
        XMLName      xml.Name     `xml:"event-list"` // XML tag
        EventList EventList // struct
}

type EventList struct {
    XMLName xml.Name `xml:"event"` // XML element tag
    Events  []EventType       // array of type struct
}

type EventType struct {
    Type            string `xml:"type,attr"`
    Id              string `xml:"id,attr"`
    GigeOutputMuxId string `xml:"id,attr"` // mux level
    AvgBitRate      string `xml:"id,attr"` // mux avg bit rate
    InstBitRate     string `xml:"id,attr"` // mux instantaneous bit rate
    Overhead        string `xml:"id,attr"` // mux overhead
}

/*
 * Format of Add Channel Event Request: (this is used only for sessions to type PUSH)
 *
 * <request id="G1168" origin="gui" destination="device" command="add" category="channel"
 * time="2015-06-25T15:11:44.595Z" protocol-version=“2.3” platform-name="neo"
 * sid="9223370601591671158"/>
 */

/*
 * Format of Add Channel Event Response:
 *
 * <response id=" G1168" origin="device" destination="gui" command="add" category="channel"
 * time="2015-06-25T15:11:44.627Z" protocol-version=“2.3” platform-name="neo" sw-
 * version="me7k.1.0.1" sw-build="1" pending-events="0">
 * <event-list>
 * <event type="alarm-deleted-event" id="1435265291353"/>
 * <event type="alarm-cleared-event" id="1435316297245" cleared- time="2015-06-
 * 26T16:38:02.513Z"/>
 * <event type="alarm-deleted-event" id="1435265291399"/>
 * <event type="alarm-cleared-event" id="1435316297244" cleared- time="2015-06-
 * 26T16:38:02.513Z"/>
 * <event type="alarm-deleted-event" id="1435265291404"/>
 * </event-list>
 * </response>
 */

/*
 * Format of Device wide subscription event request. Note: Bit rate events are NOT device wide.
 *
 * <request id="G1040" origin="gui" destination="device" command="add" category="subscription"
 * time="2016-02-08T18:16:29.271-06:00" protocol-2.3version=“2.3” platform-name="neo"
 * sid="1454976988853">
 * <path>
 * <farmer id="ME-7000-2"/>
 * </path>
 * <event-list>
 * <event type="alarm-settings-event"/>
 * <event type="configuration-event"/>
 * <event type="db-status-event"/>
 * <event type="heartbeat-event"/>
 * <event type="schedule-notification-event"/>
 * <event type="security-event"/>
 * <event type="license-event"/>
 * <event type="alarm-added-event"/>
 * <event type="alarm-deleted-event"/>
 * <event type="alarm-cleared-event"/>
 * <event type="sw-update-event"/>
 * </event-list>
 * </request>
 */

type DeviceRequest struct {
    XMLName xml.Name   `xml:"request"`
    Id          string `xml:"id,attr"`
    Origin      string `xml:"origin,attr"`
    Destination string `xml:"destination,attr"`
    Command     string `xml:"command,attr"`
    Category    string `xml:"category,attr"`
    Time        string `xml:"time,attr"`
    Version     string `xml:"protocol-version,attr"`
    Platform    string `xml:"platform-name,attr"`
    SessionId   string `xml:"sid,attr"`
    DevicePath  DevicePath   // struct
    DeviceEvent DeviceEvent  // struct
}

type DevicePath struct {
        XMLName   xml.Name     `xml:"path"` // XML tag
        FarmerId  string       `xml:"farmer"`
}

type DeviceEvent struct {
        XMLName   xml.Name  `xml:"event-list"` // XML tag
        EventType  string   `xml:"event"`
}

/*
 * Format of Subscription Event response
 *
 * <response id="G1040" origin="device" destination="gui" command="add" category="subscription"
 * time="2016-02-09T00:16:29.464Z" protocol-version=“2.3” platform-name="neo" sw-
 * version="me7k.2.2.0" sw-build="1">
 * <reason error-code="OK"><![CDATA[succeeded.]]></reason>
 * </response>
 */

//type KeepAlive struct { // not used
//    SessionId string `xml:"sid id,attr"`
//}

/*
 * General format for request:
 *
 * <request id="any-id" origin="client" destination="device" command="get" category="config"
 * time="2015-03-09T17:38:09.783-06:00" protocol- version=“2.3” platform-name="neo"
 * sid="unique-session-id" [additional attributes]>
 * <path>
 * <elements defining the path to target object>
 * </path>
 * <elements and attributes for the target object/>
 * </request>
 */

/*func LoginMarshal() {

    fmt.Printf ("LoginMarshal - enter...\n")

    // var g_sessionId // session id returned by the me7k for use in all subsequent request messages

  	v := &LoginRequest{Id: "beacham", Origin: "transcoder-collector"}
    v.Destination = "device"
  	v.Command = "add"
    v.Category = "login"
    v.Version = "2.3"
    v.Platform = "neo"
    u := &User{Name: "Admin", Password: "", Type: "push"}
    v.User = *u

	output, err := xml.Marshal(v)
  	if err != nil {
  		fmt.Printf("LoginMarshal - error: %v\n", err)
  	}

    os.Stdout.Write([]byte(xml.Header))
  	os.Stdout.Write(output)

    fmt.Printf ("LoginMarshal - ...exit\n")

    //return XMLBody

}*/

/*
 * <request id="G1037" origin="gui" destination="device" command="remove" category="login"
 * time="2015-03-09T17:38:29.783-06:00" protocol-version=“2.3” platform-name="neo"
 * sid="949098745790">
 * </request>
 */

type RemoveLoginRequest struct {
    XMLName xml.Name   `xml:"request"`
    Id          string `xml:"id,attr"`
    Origin      string `xml:"origin,attr"`
    Destination string `xml:"destination,attr"`
    Command     string `xml:"command,attr"`
    Category    string `xml:"category,attr"`
    Time        string `xml:"time,attr"`
    Version     string `xml:"protocol-version,attr"`
    Platform    string `xml:"platform-name,attr"`
    SessionId   string `xml:"sid,attr"`
}

/*
 * <response id="G1037" origin="gui" destination="device" command="remove" category="login"
 * time="2015-03-09T17:38:29.783-06:00" protocol- version=“2.3” platform-name="neo"
 * sid="949098745790">
 * <reason err-code="OK">succeeded</reason>
 * </response>
 */

type RemoveLoginResponse struct {
    XMLName xml.Name   `xml:"request"`
    Id          string `xml:"id,attr"`
    Origin      string `xml:"origin,attr"`
    Destination string `xml:"destination,attr"`
    Command     string `xml:"command,attr"`
    Category    string `xml:"category,attr"`
    Time        string `xml:"time,attr"`
    Version     string `xml:"protocol-version,attr"`
    Platform    string `xml:"platform-name,attr"`
    SessionId   string `xml:"sid,attr"`
    Reason      Reason `xml:"reason"`// struct
}

type Reason struct {
    XMLName     xml.Name `xml:"reason"`
    ErrCode     string   `xml:"err-code,attr"`
    ErrResp     string   // not sure how to encode the "succeeded" string in the XML
}

//
// Use httpClient to actually send the request
//
// input:
//
// req of type *http.Request
//
// return:
//
// string - the body of the response if no error or "" if error
// error  - a string containing the error response.StatusCode
//

func SendHTTPRequest (req *http.Request) (string, error){

    fmt.Println("SendHTTPRequest - enter...")

    response, err := httpClient.Do(req)
    if err != nil && response == nil {
        fmt.Println("SendHTTPRequest - Error sending request to API endpoint. %+v\n", err)
        return ("", fmt.Errorf("SendHTTPRequest - Status error: %v", response.StatusCode))
    } else {
        // Close the connection to reuse it
        defer response.Body.Close()

        // Let's check if the work actually is done
        // We have seen inconsistencies even when we get 200 OK response
        body, err := ioutil.ReadAll(response.Body)
        if err != nil {
            fmt.Println("SendHTTPRequest - Couldn't parse response body. %+v", err)
            return ("", fmt.Errorf("SendHTTPRequest - Status error: %v", response.StatusCode))
        }

        log.Println("SendHTTPRequest - Response Body:\n", string(body))

    }

    fmt.Println("SendHTTPRequest - ...exit")
    return string(body), error
}

//
// Prepare the body of the "remove login" request going to the server
//

func PrepareBody(body []byte) *http.Request {

    fmt.Println("PrepareBody - enter...")

    //req, err := http.NewRequest("POST", endPoint, strings.NewReader(output)) // fails
    //req, err := http.NewRequest("POST", g_EndPoint, bytes.NewBuffer([]byte(output)))
    req, err := http.NewRequest("POST", g_EndPoint, bytes.NewBuffer([]byte(body)))
    //req, err := http.NewRequest("POST", endPoint, bytes.NewBuffer([]byte("Post this data"))) // works
    if err != nil {
        log.Fatalf("PrepareBody - Error Occured on httpNewRequest. %+v\n", err)
    }
    //req.Header.Set("Content-Type", "application/x-www-form-urlencoded") // old
    //req.Header.Add("Content-Type", "application/xml; charset=utf-8")
    req.Header.Add("Content-Type", "text/xml; charset=utf-8")

    // Dump the prepared request going to the server
    dump, err := httputil.DumpRequestOut(req, true)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("PrepareBody - dump:%q", dump)
    fmt.Println()

    fmt.Println("PrepareBody - ...exit")

    return req
}

func GetEventReq (req * EventRequest) {

    fmt.Println("GetEventReq - enter...")

    output, err := xml.Marshal(req)
  	if err != nil {
  		fmt.Printf("GetEventReq - Marshal error on Remove Request: %v\n", err)
        // do you want to fatal? we could since this is the last action and we can recover from a failed login removal anyway
        // do you want to return error?
  	}

    fmt.Printf("GetEventReq - URL: %s \n", g_EndPoint)
    fmt.Println()
    fmt.Printf("GetEventReq - XML header: \n")
    os.Stdout.Write([]byte(xml.Header))
    fmt.Println()
    fmt.Println("GetEventReq - XML body for request:")
  	os.Stdout.Write(output)
    fmt.Println()

    fmt.Println("GetEventReq - calling PrepareBody:")
    r := PrepareBody(output)

    fmt.Println("GetEventReq - calling SendHTTPRequest:")
    SendHTTPRequest(r)

    //
    // Unmarshal the response
    //

    rsp := LoginResponse{}
    xml.Unmarshal(body, &rsp)
    log.Println("main - Unmarshal Login Response: ", rsp)
    log.Println("main - Unmarshal Login Response body - Id: ", rsp.Id)
    log.Println("main - Unmarshal Login Response body - Origin: ", rsp.Origin)
    log.Println("main - Unmarshal Login Response body - Destination: ", rsp.Destination)
    log.Println("main - Unmarshal Login Response body - Command: ", rsp.Command)
    log.Println("main - Unmarshal Login Response body - Category: ", rsp.Category)
    log.Println("main - Unmarshal Login Response body - Time: ", rsp.Time)
    log.Println("main - Unmarshal Login Response body - Version: ", rsp.Version)
    log.Println("main - Unmarshal Login Response body - Platform: ", rsp.Platform)
    log.Println("main - Unmarshal Login Response body - SwVersion: ", rsp.SwVersion)
    log.Println("main - Unmarshal Login Response body - SwBuild: ", rsp.SwBuild)
    log.Println("main - Unmarshal Login Response session - SessionId: ", rsp.Session.SessionId)
    log.Println("main - Unmarshal Login Response session - Type: ", rsp.Session.Type)
    log.Println("main - Unmarshal Login Response session - ActivityTimeout: ", rsp.Session.ActivityTimeout)
    log.Println("main - Unmarshal Login Response session - AuthMethod: ", rsp.Session.AuthMethod)
    log.Println("main - Unmarshal Login Response session - FarmerId: ", rsp.Session.FarmerId)
    log.Println("main - Unmarshal Login Response session - ClientIp: ", rsp.Session.ClientIp)
    log.Println("main - Unmarshal Login Response session - Warning: ", rsp.Session.Warning)

    fmt.Println("GetEventReq - ...exit")
}

/*
 * Note: This request is blocked by the Controller and as events are generated, they
 * are immediately written to the socket output stream. The caller receives responses by reading the
 * information from the underlying client socket. A client must monitor the socket input stream to read
 * the events as they arrive. The call remains blocked on the device until the session is terminated
 *
* It looks like you have to read the socket via low level go io.ReadFull(sock, data[]). Oh boy.
 */

func AddChannelEventReq(req * EventRequest) {
    fmt.Println("AddChannelEventReq - enter...")

    //dataLen := 256 // number of bytes to read
    //data := make([dataLen]byte)
    //err := io.ReadFull(sock, data)
    //fmt.Println("AddChannelEventReq - read 256 byes: ", data)

    fmt.Println("AddChannelEventReq - ...exit")
}

//
// Clean up - Remove Bit Rate Subscription Request
//

func RemoveBitRateReq(req *BitRateRequest) {

    fmt.Println("RemoveBitRateReq - enter...")

    //
    // Configure "bitrate subscription" request data - MAKE A FUNCTION
    //

    fmt.Println("main - configuring the bitrate subscription request")

    b := &BitRateRequest{Id: "beacham", Origin: "transcoder-collector"}
    b.Destination = "device"
  	b.Command = "remove" // note well
    b.Category = "subscription"
    b.Version = "2.1"
    b.Platform = "neo"
    ts := time.Now()
    b.Time = ts.String()
    b.SessionId = g_SessionId

    //
    // Subscribe to bit rate events at the MUX level
    //

    pMux := &Path{Farmer: Farmer{FarmerId: "ME-7000-2"}, Board: Board{BoardId: "4"}, GigeLine: GigeLine{GigeLineId: "4/3"}, GigeOutputMux: GigeOutputMux{GigeOutputMuxId: "0000"}}
    eMux := &EventBitRate{Type: "bit-rate-event", GetStreams: "true", GetStdDev: "true", GetInstBr: "true", GetAvgBr: "false"}

    //
    // Subscribe to bit rate events at the PROGRAM level
    //
    //p := &Path{FarmerId: "ME-7000-2", BoardId: "4", GigeLineId: "4/3", GigeOutputMuxId: "0000", OutputProgramId: ""}
    //e := &Event{EventType: "bit-rate-event", GetStreams: "true", GetStdDev: "true", GetInstBr: "true", GetAvgBr: "true", GetVideoInfo: "true", GetAudioInfo: "true"}

    b.Path = *pMux
    b.Event.EventBitRate = *eMux

    output, err := xml.Marshal(b)
  	if err != nil {
  		fmt.Println("main1 - Marshal error on Subscription Event Request: %v", err)
  	}

    //
    // Post process the "Path" XML because golang does NOT support marshaling self close tag at this time
    //

    data := string(output)
    reg := regexp.MustCompile("></farmer>")     // search for explicit close tag
    data = reg.ReplaceAllString(data,"/>")       // replace explicit close tag with with self close tag
    fmt.Println("farmer data ************************", data)
    fmt.Println()

    reg = regexp.MustCompile("></board>")     // search for explicit close tag
    data = reg.ReplaceAllString(data,"/>")       // replace explicit close tag with with self close tag
    fmt.Println("board data ************************", data)
    fmt.Println()

    reg = regexp.MustCompile("></gige-line>")     // search for explicit close tag
    data = reg.ReplaceAllString(data,"/>")       // replace explicit close tag with with self close tag
    fmt.Println("gige-line data ************************", data)
    fmt.Println()

    reg = regexp.MustCompile("></gige-output-mux>")     // search for explicit close tag
    data = reg.ReplaceAllString(data,"/>")       // replace explicit close tag with with self close tag
    fmt.Println("gige-output-mux data ************************", data)
    fmt.Println()

    /*reg = regexp.MustCompile("></output-program>")     // search for explicit close tag
    data = reg.ReplaceAllString(data,"/>")       // replace explicit close tag with with self close tag
    fmt.Println("output-program data ************************", data)
    fmt.Println()
    */

    //
    // Post process the "Event" XML because golang does NOT support marshaling self close tag at this time
    //

    reg = regexp.MustCompile("></event>")     // search for explicit close tag
    data = reg.ReplaceAllString(data,"/>")    // replace explicit close tag with with self close tag
    fmt.Println("type data ************************", data)
    fmt.Println()

    //
    //
    //

    fmt.Printf("RemoveBitRateReq - URL: %s \n", g_EndPoint)
    fmt.Println()
    fmt.Printf("RemoveBitRateReq - XML header: \n")
    os.Stdout.Write([]byte(xml.Header))
    fmt.Println()
    fmt.Println("RemoveBitRateReq - XML body for remove request:")
  	os.Stdout.Write(output)
    fmt.Println()

    r := PrepareBody(output)
    SendHTTPRequest(r)

    fmt.Println("RemoveBitRateReq - ...exit")

}

//
// Clean up - Remove Login Request
//

func RemoveLoginReq(req *RemoveLoginRequest) {

    fmt.Println("RemoveLoginReq - enter...")

    output, err := xml.Marshal(req)
  	if err != nil {
  		fmt.Printf("RemoveLoginReq - Marshal error on Remove Request: %v\n", err)
        // do you want to fatal? we could since this is the last action and we can recover from a failed login removal anyway
        // do you want to return error?
  	}

    fmt.Printf("RemoveLoginReq - URL: %s \n", g_EndPoint)
    fmt.Println()
    fmt.Printf("RemoveLoginReq - XML header: \n")
    os.Stdout.Write([]byte(xml.Header))
    fmt.Println()
    fmt.Println("RemoveLoginReq - XML body for remove request:")
  	os.Stdout.Write(output)
    fmt.Println()

    r := PrepareBody(output)
    SendHTTPRequest(r)

    fmt.Println("RemoveLoginReq - ...exit")

}

//
// Main
//

func main() {

    fmt.Printf ("main - enter...\n")

    //var sid string // Session Id

    //var endPoint string = "http://localhost:8080/doSomething"
    //var endPoint string = "https://10.77.6.12" // real Me7k in San Diego
    //var endPoint string = "https://www.arris.com" // real Me7k (name is configured in /etc/host to map to 10.77.6.12)
    var endPoint string = "https://10.10.55.163/neoreq/" // real Me7k in horsham
    //var endPoint string = "http://192.168.0.28:8080" // local server for testing

    //var XMLBody = bytes.NewBuffer()

    //LoginMarshal()
    //
    // Configure "login" request data
    //
    v := &LoginRequest{Id: "beacham", Origin: "transcoder-collector"}
    v.Destination = "device"
  	v.Command = "add"
    v.Category = "login"
    v.Version = "2.1"
    v.Platform = "neo"
    t := time.Now()
    v.Time = t.String()
    u := &User{Name: "Admin", Password: "", Type: "pull"} // note well. pull for get events. push for add channel
    v.User = *u

	output, err := xml.Marshal(v)
  	if err != nil {
  		fmt.Printf("main - Marshal error on Login Request: %v\n", err)
  	}

    fmt.Printf("main - URL: %s \n", endPoint)
    fmt.Printf("\n")
    fmt.Printf("main - XML header: \n")
    os.Stdout.Write([]byte(xml.Header))
    fmt.Printf("\n")
    fmt.Printf("main - XML body: \n")
  	os.Stdout.Write(output)
    fmt.Printf("\n\n")

    //
    // Prepare the body of the "login" request going to the server
    //
    //req, err := http.NewRequest("POST", endPoint, strings.NewReader(output)) // fails
    req, err := http.NewRequest("POST", endPoint, bytes.NewBuffer([]byte(output)))
    //req, err := http.NewRequest("POST", endPoint, bytes.NewBuffer([]byte("Post this data"))) // works
    if err != nil {
        log.Fatalf("main - Error Occured on httpNewRequest. %+v\n", err)
    }
    //req.Header.Set("Content-Type", "application/x-www-form-urlencoded") // old
    //req.Header.Add("Content-Type", "application/xml; charset=utf-8")
    req.Header.Add("Content-Type", "text/xml; charset=utf-8")

    // Dump the prepared request going to the server
    dump, err := httputil.DumpRequestOut(req, true)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("%q\n", dump)
    fmt.Printf("\n")

    //
    // Use httpClient to actually send the request
    //

    response, err := httpClient.Do(req)
    if err != nil && response == nil {
        log.Fatalf("main - Error sending request to API endpoint. %+v\n", err)
    } else {
        // Close the connection to reuse it
        defer response.Body.Close()

        // Let's check if the work actually is done
        // We have seen inconsistencies even when we get 200 OK response
        body, err := ioutil.ReadAll(response.Body)
        if err != nil {
            log.Fatalf("main - Couldn't parse response body. %+v", err)
        }

        log.Println("main - Response Body:\n", string(body))

        // make sure you extract the sid now as you need it for subsequent messages

        rsp := LoginResponse{}
        xml.Unmarshal(body, &rsp)
        log.Println("main - Unmarshal Login Response: ", rsp)
        log.Println("main - Unmarshal Login Response body - Id: ", rsp.Id)
        log.Println("main - Unmarshal Login Response body - Origin: ", rsp.Origin)
        log.Println("main - Unmarshal Login Response body - Destination: ", rsp.Destination)
        log.Println("main - Unmarshal Login Response body - Command: ", rsp.Command)
        log.Println("main - Unmarshal Login Response body - Category: ", rsp.Category)
        log.Println("main - Unmarshal Login Response body - Time: ", rsp.Time)
        log.Println("main - Unmarshal Login Response body - Version: ", rsp.Version)
        log.Println("main - Unmarshal Login Response body - Platform: ", rsp.Platform)
        log.Println("main - Unmarshal Login Response body - SwVersion: ", rsp.SwVersion)
        log.Println("main - Unmarshal Login Response body - SwBuild: ", rsp.SwBuild)
        log.Println("main - Unmarshal Login Response session - SessionId: ", rsp.Session.SessionId)
        log.Println("main - Unmarshal Login Response session - Type: ", rsp.Session.Type)
        log.Println("main - Unmarshal Login Response session - ActivityTimeout: ", rsp.Session.ActivityTimeout)
        log.Println("main - Unmarshal Login Response session - AuthMethod: ", rsp.Session.AuthMethod)
        log.Println("main - Unmarshal Login Response session - FarmerId: ", rsp.Session.FarmerId)
        log.Println("main - Unmarshal Login Response session - ClientIp: ", rsp.Session.ClientIp)
        log.Println("main - Unmarshal Login Response session - Warning: ", rsp.Session.Warning)

        g_SessionId = rsp.Session.SessionId // save session id as global for use with events

        //
        // Does Unmarshal return an error?
        //
  	    /*if err != nil {
  		    fmt.Printf("main - Unmarshal error: %v\n", err)
  	     }*/

    }

    //
    // Configure "device subscription" request data
    //

    //
    // Configure "bitrate subscription" request data - MAKE A FUNCTION
    //

    fmt.Println("main - configuring the bitrate subscription request")

    b := &BitRateRequest{Id: "beacham", Origin: "transcoder-collector"}
    b.Destination = "device"
  	b.Command = "add" // note well
    b.Category = "subscription"
    b.Version = "2.1"
    b.Platform = "neo"
    ts := time.Now()
    b.Time = ts.String()
    b.SessionId = g_SessionId

    //
    // Subscribe to bit rate events at the MUX level
    //

    pMux := &Path{Farmer: Farmer{FarmerId: "ME-7000-2"}, Board: Board{BoardId: "4"}, GigeLine: GigeLine{GigeLineId: "4/3"}, GigeOutputMux: GigeOutputMux{GigeOutputMuxId: "0000"}}
    eMux := &EventBitRate{Type: "bit-rate-event", GetStreams: "true", GetStdDev: "true", GetInstBr: "true", GetAvgBr: "false"}

    //
    // Subscribe to bit rate events at the PROGRAM level
    //
    //p := &Path{FarmerId: "ME-7000-2", BoardId: "4", GigeLineId: "4/3", GigeOutputMuxId: "0000", OutputProgramId: ""}
    //e := &Event{EventType: "bit-rate-event", GetStreams: "true", GetStdDev: "true", GetInstBr: "true", GetAvgBr: "true", GetVideoInfo: "true", GetAudioInfo: "true"}

    b.Path = *pMux
    b.Event.EventBitRate = *eMux

	output, err = xml.Marshal(b)
  	if err != nil {
  		fmt.Println("main1 - Marshal error on Subscription Event Request: %v", err)
  	}

    //
    // Post process the "Path" XML because golang does NOT support marshaling self close tag at this time
    //

    data := string(output)
    reg := regexp.MustCompile("></farmer>")     // search for explicit close tag
    data = reg.ReplaceAllString(data,"/>")       // replace explicit close tag with with self close tag
    fmt.Println("farmer data ************************", data)
    fmt.Println()

    reg = regexp.MustCompile("></board>")     // search for explicit close tag
    data = reg.ReplaceAllString(data,"/>")       // replace explicit close tag with with self close tag
    fmt.Println("board data ************************", data)
    fmt.Println()

    reg = regexp.MustCompile("></gige-line>")     // search for explicit close tag
    data = reg.ReplaceAllString(data,"/>")       // replace explicit close tag with with self close tag
    fmt.Println("gige-line data ************************", data)
    fmt.Println()

    reg = regexp.MustCompile("></gige-output-mux>")     // search for explicit close tag
    data = reg.ReplaceAllString(data,"/>")       // replace explicit close tag with with self close tag
    fmt.Println("gige-output-mux data ************************", data)
    fmt.Println()

    /*reg = regexp.MustCompile("></output-program>")     // search for explicit close tag
    data = reg.ReplaceAllString(data,"/>")       // replace explicit close tag with with self close tag
    fmt.Println("output-program data ************************", data)
    fmt.Println()
*/

    //
    // Post process the "Event" XML because golang does NOT support marshaling self close tag at this time
    //

    reg = regexp.MustCompile("></event>")     // search for explicit close tag
    data = reg.ReplaceAllString(data,"/>")    // replace explicit close tag with with self close tag
    fmt.Println("type data ************************", data)
    fmt.Println()

    //

    fmt.Printf("main - URL: %s \n", g_EndPoint)
    fmt.Println()
    fmt.Printf("main - XML header: \n")
    os.Stdout.Write([]byte(xml.Header))
    fmt.Printf("\n")
    fmt.Println("main - XML body for subscription event request:")
  	os.Stdout.Write(output) //sb data right?
    fmt.Println()

    //
    // Prepare the body of the "subscription event" request going to the server
    //

    fmt.Println("main - preparing the bitrate subscription request")

    //req, err := http.NewRequest("POST", endPoint, strings.NewReader(output)) // fails
    req, err = http.NewRequest("POST", g_EndPoint, bytes.NewBuffer([]byte(output)))
    //req, err := http.NewRequest("POST", endPoint, bytes.NewBuffer([]byte("Post this data"))) // works
    if err != nil {
        log.Fatalf("main - Error Occured on httpNewRequest. %+v\n", err)
    }
    //req.Header.Set("Content-Type", "application/x-www-form-urlencoded") // old
    //req.Header.Add("Content-Type", "application/xml; charset=utf-8")
    req.Header.Add("Content-Type", "text/xml; charset=utf-8")

    // Dump the prepared request going to the server
    dump, err = httputil.DumpRequestOut(req, true)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("HTTP subscription event request being sent to server: %q", dump)
    fmt.Println()

    //
    // Use httpClient to actually send the request - should be a separate function
    //

    fmt.Println("main - send the bitrate subscription request")

    response, err = httpClient.Do(req)
    if err != nil && response == nil {
        log.Fatalf("main - Error sending request to API endpoint. %+v\n", err)
    } else {
        // Close the connection to reuse it
        defer response.Body.Close()

        // Let's check if the work actually is done
        // We have seen inconsistencies even when we get 200 OK response
        body, err := ioutil.ReadAll(response.Body)
        if err != nil {
            log.Fatalf("main - Couldn't parse response body. %+v", err)
        }

        log.Println("main - Response Body:\n", string(body))

    }

    //
    // Now we need to listen for bit rate events sent to us by the "device". There are two approaches:
    // 1) real time using an add channel event req/rsp sequence or
    // 2) bulk using get events with a variable number of events per get.
    //

    a := &EventRequest{Id: "beacham", Origin: "transcoder-collector"}
    a.Destination = "device"
  	a.Command = "get"      // note well or "add"
    a.Category = "event" // note well or "channel"
    a.Version = "2.1"
    a.Platform = "neo"
    quantum := time.Now()
    a.Time = quantum.String()
    a.SessionId = g_SessionId

    fmt.Println("main - Subscription Bitrate Event listen loop - enter...")

    timeChan := time.NewTimer(time.Minute).C
    tickChan := time.NewTicker(time.Millisecond * 500).C
    exitChan := make(chan bool) // create timer in concurrent channel

    go func() { // set a timer to expire in 30 mins of event collection
        time.Sleep(time.Minute * 1)
        exitChan <- true // when timer expires, set exitChan to true so we break out of the loop below

    }() // end in line timer fx

    i := 0
    killme := false

    for {

        GetEventReq(a)
        /*
        if err != nil { // exit if network error or can't understand response body
            log.Fatalf("main - Couldn't parse response body. %+v", err)
            fmt.Printlin("main - Couldn't parse response body. %+v", err")
            break
        }
        */

        select {

        case <- timeChan:
            fmt.Println("Timer expired") // for testing. explicitly exit after 30 mins of event collection
        case <- tickChan:
            fmt.Println("Ticker ticked: ", i) // print time stamp
            i++
        case <- exitChan:
            fmt.Println("Exit select")
            killme = true
            break // exit select

        } // end select
        if killme == true {
            fmt.Println("Exit event loop")
            break // exit for
        }
    } // end for

    //timeChan.Stop() // I guess exitChan terminates the timers?
    //tickChan.Stop()

    fmt.Println("main - Subscription Bitrate Event listen loop - ...exit")

    //
    // Clean up - Remove Bitrate Subscription Event
    //

    RemoveBitRateReq(b)

    //
    // Clean up by removing login session
    //

    // Prepare RemoveLoginRequest message content

    r := &RemoveLoginRequest{Id: "beacham", Origin: "transcoder-collector"}
    r.Destination = "device"
  	r.Command = "remove"
    r.Category = "login"
    r.Version = "2.1"
    r.Platform = "neo"
    q := time.Now()
    r.Time = q.String()
    r.SessionId = g_SessionId

    RemoveLoginReq(r)

    fmt.Println ("main - ...exit")

}
