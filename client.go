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
)

var (
    httpClient *http.Client
)

const (
    MaxIdleConnections int = 20
    RequestTimeout     int = 5
)

const (
        // A generic XML header suitable for use with the output of Marshal.
        // This is not automatically added to any output of this package,
        // it is provided as a convenience.
        Header = `<?xml version="1.0" encoding="UTF-8"?>` + "\n"
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

/*
 * Format of Subscription (bit rate) request:
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
 * get-avg-br="true" get-video-info="true" get- audio-info="true"/>
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
    Time        string `xml:"time,attr"`     // how do i get a timestamp?
    Version     string `xml:"protocol-version,attr"`
    Platform    string `xml:"platform-name,attr"`
    SessionId   string `xml:"sid,attr"`
    Path  Path   // struct
    Event Event  // struct
}

type Path struct {
        XMLName   xml.Name     `xml:"path"` // XML tag
        FarmerId  string       `xml:"farmer id,attr"`
        BoardId   string       `xml:"board id,attr`
        GigeLineId string      `xml:"gige-line id,attr"`
        GigeOutputMuxId string `xml:"gige-output-mux id,attr"`
        OutputProgramId string `xml:"output-program id,attr"`
}

type Event struct {
        XMLName   xml.Name  `xml:"event-list"` // XML tag
        EventType  string   `xml:"farmer id,attr"`
        GetStreams string   `xml:"board id,attr`
        GetStdDev string    `xml:"gige-line id,attr"`
        GetInstBr string    `xml:"gige-output-mux id,attr"`
        GetAvgBr string     `xml:"output-program id,attr"`
        GetVideoInfo string `xml:"output-program id,attr"`
        GetAudioInfo string `xml:"output-program id,attr"`
}

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
    Time        string `xml:"time,attr"`     // how do i get a timestamp?
    Version     string `xml:"protocol-version,attr"`
    Platform    string `xml:"platform-name,attr"`
    SessionId   string `xml:"sid,attr"`
    DevicePath  DevicePath   // struct
    DeviceEvent DeviceEvent  // struct
}

type DevicePath struct {
        XMLName   xml.Name     `xml:"path"` // XML tag
        FarmerId  string       `xml:"farmer id,attr"`
}

type DeviceEvent struct {
        XMLName   xml.Name  `xml:"event-list"` // XML tag
        EventType  string   `xml:"farmer id,attr"`
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

type KeepAlive struct {
    SessionId string `xml:"sid id,attr"`
}

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
    u := &User{Name: "Admin", Password: "", Type: "push"}
    v.User = *u

	output, err := xml.Marshal(v)
  	if err != nil {
  		fmt.Printf("main - Marshal error: %v\n", err)
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
        log.Fatalf("main - Error Occured. %+v\n", err)
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

  	    /*if err != nil {
  		    fmt.Printf("main - UnMarshal error: %v\n", err)
  	     }*/

    }

    //
    // Configure "device subscription" request data
    //

    fmt.Printf ("main - ...exit\n")

}
